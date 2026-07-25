package performance_analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetDashboardData(ctx context.Context, from, to time.Time) (SummaryDashboard, error) {
	summary, err := s.repo.GetSummary(ctx, from, to)
	if err != nil {
		return SummaryDashboard{}, err
	}

	return SummaryDashboard{
		DateRange: DateRangeConfig{
			StartDate: from,
			EndDate:   to.Add(-time.Nanosecond),
		},
		TotalRevenue:        summary.totalRevenue,
		AverageRating:       summary.averageRating,
		AverageProfitMargin: summary.averageProfitMargin,
		TotalReviews:        int(summary.totalReviews),
	}, nil
}

func (s *Service) GetMenuItems(
	ctx context.Context,
	from, to time.Time,
	sortBy string,
	performanceCategory string,
) (MenuItemsResponse, error) {
	window := to.Sub(from)
	if window <= 0 {
		return MenuItemsResponse{}, nil
	}
	previousTo := from
	previousFrom := previousTo.Add(-window)

	rows, err := s.repo.GetTopItems(ctx, from, to, previousFrom, previousTo, sortBy, 5)
	if err != nil {
		return MenuItemsResponse{}, err
	}

	totalUnits, err := s.repo.GetTotalUnits(ctx, from, to)
	if err != nil {
		return MenuItemsResponse{}, err
	}

	items := buildMenuItems(rows, totalUnits, performanceCategory)
	return MenuItemsResponse{
		DateRange: DateRangeConfig{
			StartDate: from,
			EndDate:   to.Add(-time.Nanosecond),
		},
		Items: items,
	}, nil
}

func buildMenuItems(rows []menuItemAggregate, totalUnits int, categoryFilter string) []MenuItem {
	if len(rows) == 0 {
		return []MenuItem{}
	}

	avgPopularity := 0.0
	avgMargin := 0.0
	for _, row := range rows {
		avgMargin += row.ContributionMargin
		if totalUnits > 0 {
			avgPopularity += (float64(row.UnitsSold) / float64(totalUnits)) * 100
		}
	}
	avgPopularity /= float64(len(rows))
	avgMargin /= float64(len(rows))

	items := make([]MenuItem, 0, len(rows))
	for _, row := range rows {
		popularity := 0.0
		if totalUnits > 0 {
			popularity = (float64(row.UnitsSold) / float64(totalUnits)) * 100
		}
		category := classifyPerformance(popularity, row.ContributionMargin, avgPopularity, avgMargin)
		if categoryFilter != "" && categoryFilter != category {
			continue
		}
		items = append(items, MenuItem{
			ID:                  row.ID,
			Name:                row.Name,
			MenuCategory:        row.MenuCategory,
			UnitsSold:           row.UnitsSold,
			PopularityIndex:     popularity,
			Revenue:             row.Revenue,
			FoodCostPercent:     row.FoodCostPercent,
			ContributionMargin:  row.ContributionMargin,
			PerformanceCategory: category,
			TrendPercent:        row.TrendPercent,
		})
	}

	return items
}

func classifyPerformance(popularity, contributionMargin, avgPopularity, avgMargin float64) string {
	switch {
	case popularity >= avgPopularity && contributionMargin >= avgMargin:
		return "star"
	case popularity >= avgPopularity && contributionMargin < avgMargin:
		return "plowhorse"
	case popularity < avgPopularity && contributionMargin >= avgMargin:
		return "puzzle"
	default:
		return "dog"
	}
}

// GetGoogleReviews pulls reviews from Google Places and stores them in Postgres.
func (s *Service) GetGoogleReviews() (*GooglePlaceAPIResponse, error) {
	apiKey := os.Getenv("GOOGLE_MAP_DEMO_API_KEY")
	placeID := os.Getenv("GOOGLE_PLACEID_LAV_API_KEY")
	fullURL := fmt.Sprintf("https://places.googleapis.com/v1/places/%s", placeID)
	req, _ := http.NewRequest("GET", fullURL, nil)
	req.Header.Add("X-Goog-Api-Key", apiKey)
	req.Header.Add("X-Goog-FieldMask", "id,displayName,rating,userRatingCount,reviews,reviewSummary")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data GooglePlaceAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if len(data.Reviews) > 0 {
		err = s.repo.SaveGoogleReviews(context.Background(), data.Reviews)
		if err != nil {
			return nil, fmt.Errorf("failed to save reviews to db: %w", err)
		}
	}

	return &data, nil
}

// GetCloverOrders pulls today's Clover orders from Clover's API.
func (s *Service) GetCloverOrders() (*CloverOrderResponse, error) {
	merchantID := os.Getenv("CLOVER_MERCHANT_MID")
	apiToken := os.Getenv("CLOVER_DEV_API_KEY")
	baseURL := "https://apisandbox.dev.clover.com/v3/merchants/"

	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	midnightMs := midnight.UnixNano() / int64(time.Millisecond)

	fullURL := fmt.Sprintf("%s%s/orders?filter=createdTime>=%d&expand=lineItems,totals&limit=100", baseURL, merchantID, midnightMs)

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+apiToken)
	req.Header.Add("accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("clover API returned error status: %d", resp.StatusCode)
	}

	var data CloverOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}
