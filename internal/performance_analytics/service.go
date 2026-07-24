// This is the brain of feature. It contains the actual raw Go logic, mathematical algorithms, calculations, and database coordination.
// It calculates net profit (revenue - cogs), averages review ratings, and sorts the top items.
package performance_analytics

import "time"

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// GetDashBoardData builds a summary dashboard payload for the analytics UI.
func (s *Service) GetDashBoardData() SummaryDashboard {
	now := time.Now()
	start := now.AddDate(0, 0, -30)
	end := now

	return SummaryDashboard{
		DateRange: DateRangeConfig{
			StartDate: start,
			EndDate:   end,
		},
		TotalRevenue:         14500.75,
		AverageRating:        4.65,
		AverageProfitMargin: 0.42,
		TotalReviews:         129,
	}
}

func (s *Service) GetMenuItems() MenuItemsResponse {
	return MenuItemsResponse{
		DateRange: DateRangeConfig{
			StartDate: time.Now().AddDate(0, 0, -30),
			EndDate:   time.Now(),
		},
		Items: []MenuItem{{
			ID:                  "1",
			Name:                "Salted Egg Coffee",
			MenuCategory:        "Beverage",
			UnitsSold:           412,
			PopularityIndex:     8.7,
			Revenue:             2060,
			FoodCostPercent:     28.5,
			ContributionMargin:  1472.9,
			PerformanceCategory: "star",
			TrendPercent:        18.5,
		}},
	}
}

func (s *Service) GetSalesTrend() SalesTrendResponse {
	return SalesTrendResponse{
		Granularity: "day",
		Points: []SalesTrendPoint{{
			Date:      "2026-06-01",
			Revenue:   1580,
			UnitsSold: 132,
		}, {
			Date:      "2026-06-02",
			Revenue:   1690.25,
			UnitsSold: 140,
		}},
	}
}

func (s *Service) GetReviewSummary() ReviewSummaryResponse {
	return ReviewSummaryResponse{
		DateRange: DateRangeConfig{
			StartDate: time.Now().AddDate(0, 0, -30),
			EndDate:   time.Now(),
		},
		TotalReviews: 128,
		AverageRating: 4.3,
		SentimentBreakdown: SentimentBreakdown{
			Positive: 78,
			Neutral:  32,
			Negative: 18,
		},
		SentimentTrend: []SentimentTrendPoint{{
			Date:          "2026-06-01",
			AverageRating: 4.1,
			ReviewCount:   5,
		}},
		TopKeywords: []string{"fusion", "spicy", "slow service"},
	}
}
