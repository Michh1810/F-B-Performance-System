// PostgreSQL aggregation logic

package performance_analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

type dashboardSummaryRow struct {
	totalRevenue        float64
	averageProfitMargin float64
	averageRating       float64
	totalReviews        int64
}

type menuItemAggregate struct {
	ID                 string
	Name               string
	MenuCategory       string
	UnitsSold          int
	Revenue            float64
	FoodCostPercent    float64
	ContributionMargin float64
	TrendPercent       float64
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetSummary(ctx context.Context, from, to time.Time) (dashboardSummaryRow, error) {
	const query = `
WITH transaction_summary AS (
  SELECT
    COALESCE(SUM(t.quantity * t.unit_price), 0) AS total_revenue,
    COALESCE(
      (SUM((t.unit_price - mi.cogs) * t.quantity) / NULLIF(SUM(t.quantity * t.unit_price), 0)) * 100,
      0
    ) AS average_profit_margin
  FROM transactions t
  JOIN menu_items mi ON mi.id = t.menu_item_id
  WHERE t.sold_at >= $1 AND t.sold_at < $2
),
review_summary AS (
  SELECT
    COALESCE(AVG(rv.rating), 0) AS average_rating,
    COALESCE(COUNT(rv.id), 0) AS total_reviews
  FROM reviews rv
  WHERE rv.created_at >= $1 AND rv.created_at < $2
)
SELECT
  ts.total_revenue,
  ts.average_profit_margin,
  rs.average_rating,
  rs.total_reviews
FROM transaction_summary ts
CROSS JOIN review_summary rs;
`

	var row dashboardSummaryRow
	if err := r.db.QueryRow(ctx, query, from, to).Scan(
		&row.totalRevenue,
		&row.averageProfitMargin,
		&row.averageRating,
		&row.totalReviews,
	); err != nil {
		return dashboardSummaryRow{}, err
	}
	return row, nil
}

func (r *Repository) GetTotalUnits(ctx context.Context, from, to time.Time) (int, error) {
	const query = `
SELECT COALESCE(SUM(quantity), 0)
FROM transactions
WHERE sold_at >= $1 AND sold_at < $2;
`

	var units int64
	if err := r.db.QueryRow(ctx, query, from, to).Scan(&units); err != nil {
		return 0, err
	}
	return int(units), nil
}

func (r *Repository) GetTopItems(
	ctx context.Context,
	from, to time.Time,
	previousFrom, previousTo time.Time,
	sortBy string,
	limit int,
) ([]menuItemAggregate, error) {
	orderBy, err := topItemsOrderBy(sortBy)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
WITH current_period AS (
  SELECT
    mi.id,
    mi.name,
    mi.category,
    SUM(t.quantity)::int AS units_sold,
    SUM(t.quantity * t.unit_price) AS revenue,
    COALESCE((mi.cogs / NULLIF(mi.current_price, 0)) * 100, 0) AS food_cost_percent,
    SUM((t.unit_price - mi.cogs) * t.quantity) AS contribution_margin
  FROM menu_items mi
  JOIN transactions t ON t.menu_item_id = mi.id
  WHERE t.sold_at >= $1 AND t.sold_at < $2
  GROUP BY mi.id, mi.name, mi.category, mi.cogs, mi.current_price
),
previous_period AS (
  SELECT
    t.menu_item_id,
    SUM(t.quantity * t.unit_price) AS previous_revenue
  FROM transactions t
  WHERE t.sold_at >= $3 AND t.sold_at < $4
  GROUP BY t.menu_item_id
)
SELECT
  cp.id::text,
  cp.name,
  cp.category,
  cp.units_sold,
  cp.revenue,
  cp.food_cost_percent,
  cp.contribution_margin,
  CASE
    WHEN COALESCE(pp.previous_revenue, 0) = 0 THEN 0
    ELSE ((cp.revenue - pp.previous_revenue) / pp.previous_revenue) * 100
  END AS trend_percent
FROM current_period cp
LEFT JOIN previous_period pp ON pp.menu_item_id = cp.id
ORDER BY %s
LIMIT $5;
`, orderBy)

	rows, err := r.db.Query(ctx, query, from, to, previousFrom, previousTo, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]menuItemAggregate, 0, limit)
	for rows.Next() {
		var item menuItemAggregate
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.MenuCategory,
			&item.UnitsSold,
			&item.Revenue,
			&item.FoodCostPercent,
			&item.ContributionMargin,
			&item.TrendPercent,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func topItemsOrderBy(sortBy string) (string, error) {
	switch sortBy {
	case "", "revenue":
		return "cp.revenue DESC", nil
	case "unitsSold":
		return "cp.units_sold DESC", nil
	case "profitMargin":
		return "cp.contribution_margin DESC", nil
	default:
		return "", fmt.Errorf("invalid sortBy %q", sortBy)
	}
}

func (r *Repository) SaveGoogleReviews(ctx context.Context, reviews []GoogleReview) error {
	const query = `
		INSERT INTO google_reviews (
			review_id, source, star, author_name, review_count, review_date, review_text
		) VALUES (
			$1, 'google', $2, $3, $4, $5, $6
		) ON CONFLICT (review_id) DO NOTHING
	`

	// Note: We use pgx.Batch to insert all reviews efficiently in one round trip.
	batch := &pgx.Batch{}
	for _, rev := range reviews {
		// Google review name looks like "places/PLACE_ID/reviews/REVIEW_ID"
		// We'll pass the full name, but keep in mind that the db schema has VARCHAR(22).
		// If Google review IDs are longer than 22 characters, PostgreSQL will return an error
		// and you might need to run an ALTER TABLE to increase the length.
		reviewID := rev.Name

		// If it has a prefix "places/.../reviews/", we can extract just the ID part
		// but typically it's still >22 chars. Let's just use the last 22 characters to ensure it fits the table constraint for now.
		if len(reviewID) > 22 {
			reviewID = reviewID[len(reviewID)-22:]
		}

		batch.Queue(query,
			reviewID,
			int(rev.Rating),
			rev.AuthorAttribution.DisplayName,
			0, // Google API doesn't return author's total review_count, so defaulting to 0
			rev.PublishTime,
			rev.Text.Text,
		)
	}

	br := r.db.SendBatch(ctx, batch)
	defer br.Close()

	if _, err := br.Exec(); err != nil {
		return err
	}

	return nil
}
