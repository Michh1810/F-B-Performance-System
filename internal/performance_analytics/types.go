package performance_analytics

import "time"

//return top-items(profitMargin), salestrend, averagerating, totalrevenue, reviewsummary and thêm dateRange.



type SummaryDashboard struct {
	DateRange DateRangeConfig `json:"date_range"`
	TotalRevenue float64 `json:"total_revenue"`
	AverageRating float64 `json:"average_rating"`
	AverageProfitMargin float64 `json:"average_profit_margin"`
	TotalReviews int `json:"total_reviews"`
}


// Date Range of Data
type DateRangeConfig struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}


// HÙNG PART
type MenuItem struct {
    ID                  string  `json:"id"`
    Name                string  `json:"name"`
    MenuCategory        string  `json:"menuCategory"`
    UnitsSold           int     `json:"unitsSold"`
    PopularityIndex     float64 `json:"popularityIndex"`
    Revenue             float64 `json:"revenue"`
    FoodCostPercent     float64 `json:"foodCostPercent"`
    ContributionMargin  float64 `json:"contributionMargin"`
    PerformanceCategory string  `json:"performanceCategory"`
    TrendPercent        float64 `json:"trendPercent"`
}

type MenuItemsResponse struct {
    DateRange DateRangeConfig `json:"dateRange"`
    Items     []MenuItem      `json:"items"`
}

type SalesTrendPoint struct {
    Date      string  `json:"date"`
    Revenue   float64 `json:"revenue"`
    UnitsSold int     `json:"unitsSold"`
}

type SalesTrendResponse struct {
    Granularity string            `json:"granularity"`
    Points      []SalesTrendPoint `json:"points"`
}

type SentimentBreakdown struct {
    Positive int `json:"positive"`
    Neutral  int `json:"neutral"`
    Negative int `json:"negative"`
}

type SentimentTrendPoint struct {
    Date          string  `json:"date"`
    AverageRating float64 `json:"averageRating"`
    ReviewCount   int     `json:"reviewCount"`
}

type ReviewSummaryResponse struct {
    DateRange          DateRangeConfig      `json:"dateRange"`
    TotalReviews       int                  `json:"totalReviews"`
    AverageRating      float64              `json:"averageRating"`
    SentimentBreakdown SentimentBreakdown   `json:"sentimentBreakdown"`
    SentimentTrend     []SentimentTrendPoint `json:"sentimentTrend"`
    TopKeywords        []string             `json:"topKeywords"`
}
