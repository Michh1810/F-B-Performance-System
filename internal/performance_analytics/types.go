package performance_analytics

type DateRangeConfig struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type SummaryDashboard struct {
	DateRange           DateRangeConfig `json:"dateRange"`
	TotalRevenue        float64         `json:"totalRevenue"`
	AverageRating       float64         `json:"averageRating"`
	AverageProfitMargin float64         `json:"averageProfitMargin"`
	TotalReviews        int             `json:"totalReviews"`
}

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
