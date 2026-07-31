package overview

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type WeeklyMetrics struct {
	TotalRevenue float64
	TotalOrders  int
	AverageRating float64
	TotalReviews int
}

func (r *Repository) GetWeeklyMetrics(ctx context.Context, from, to time.Time) (WeeklyMetrics, error) {
	var m WeeklyMetrics

	// Calculate total revenue and order count
	err := r.db.QueryRow(ctx, `
		SELECT 
			COALESCE(SUM(quantity * unit_price), 0),
			COUNT(DISTINCT sold_at)
		FROM transactions 
		WHERE sold_at >= $1 AND sold_at < $2
	`, from, to).Scan(&m.TotalRevenue, &m.TotalOrders)
	
	if err != nil {
		return m, err
	}

	// Calculate sentiment
	err = r.db.QueryRow(ctx, `
		SELECT 
			COALESCE(AVG(star), 0),
			COUNT(review_id)
		FROM google_reviews 
		WHERE review_date >= $1 AND review_date < $2
	`, from, to).Scan(&m.AverageRating, &m.TotalReviews)

	if err != nil {
		return m, err
	}

	return m, nil
}
