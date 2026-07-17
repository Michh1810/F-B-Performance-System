// This is the brain of feature. It contains the actual raw Go logic, mathematical algorithms, calculations, and database coordination.
//
//	It calculates net profit (`revenue - cogs`), averages review ratings, and sorts the top items.
package performance_analytics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Service struct { // declare an empty structure that represents business tool, when connect real data base, we must update this
	//db *pgx.Pool
	// redis *redis.Client
}

func NewService() *Service {
	return &Service{}
}

// You place (s *Service) right before the function name.
// This is Go's version of writing def get_data(self): in Python. // This function must build and return a "SummaryDashboard" struct
func (s *Service) GetDashBoardData() SummaryDashboard {
	now := time.Now()
	start := now.AddDate(0, 0, -30)
	end := now

	data := SummaryDashboard{
		DateRange: DateRangeConfig{
			StartDate: start,
			EndDate:   end},
		//  this is mock JSON Data
		TotalRevenue:        14500.75,
		AverageRating:       4.65,
		AverageProfitMargin: 0.42,
		TotalReviews:        129,
	}

	return data
}

func (s *Service) GetMenuItems() MenuItemsResponse {
	return MenuItemsResponse{
		DateRange: DateRangeConfig{
			StartDate: time.Now().AddDate(0, 0, -30),
			EndDate:   time.Now(),
		},
		Items: []MenuItem{
			{
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
			},
		},
	}
}

func (s *Service) GetSalesTrend() SalesTrendResponse {

	return SalesTrendResponse{
		Granularity: "day",
		Points: []SalesTrendPoint{
			{
				Date:      "2026-06-01",
				Revenue:   1580,
				UnitsSold: 132,
			},
			{
				Date:      "2026-06-02",
				Revenue:   1690.25,
				UnitsSold: 140,
			},
		},
	}
}

func (s *Service) GetGoogleReviews() (*GooglePlaceAPIResponse, error) {
	apiKey := os.Getenv("GOOGLE_MAP_DEMO_API_KEY")
	placeID := "ChIJH_LybIPhwogRytz9jhj7d14"
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

	//Decode JSON into created struct in types.go
	var data GooglePlaceAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	// return the structured GO data
	return &data, nil
}

func (s *Service) GetReviewSummary() ReviewSummaryResponse {

	return ReviewSummaryResponse{
		DateRange: DateRangeConfig{
			StartDate: time.Now().AddDate(0, 0, -30),
			EndDate:   time.Now(),
		},

		TotalReviews:  128,
		AverageRating: 4.3,

		SentimentBreakdown: SentimentBreakdown{
			Positive: 78,
			Neutral:  32,
			Negative: 18,
		},

		SentimentTrend: []SentimentTrendPoint{
			{
				Date:          "2026-06-01",
				AverageRating: 4.1,
				ReviewCount:   5,
			},
		},

		TopKeywords: []string{
			"fusion",
			"spicy",
			"slow service",
		},
	}
}
