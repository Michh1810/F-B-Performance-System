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

// Struct for Google Review JSON
type DisplayName struct {
	Text         string `json:"text"`
	LanguageCode string `json:"languageCode"`
}

type LocalizedText struct {
	Text         string `json:"text"`
	LanguageCode string `json:"languageCode"`
}

type AuthorAttribution struct {
	DisplayName string `json:"displayName"`
	Uri         string `json:"uri"`
	PhotoUri    string `json:"photoUri"`
}

type GoogleReview struct {
	Name                           string            `json:"name"`
	RelativePublishTimeDescription string            `json:"relativePublishTimeDescription"`
	Rating                         float64           `json:"rating"`
	Text                           LocalizedText     `json:"text"`
	OriginalText                   LocalizedText     `json:"originalText"`
	AuthorAttribution              AuthorAttribution `json:"authorAttribution"`
	PublishTime                    string            `json:"publishTime"`
}
type ReviewSummary struct {
	Text LocalizedText `json:"text"`
}

// Parent object for Google reviews
type GooglePlaceAPIResponse struct {
	ID              string         `json:"id"`
	DisplayName     DisplayName    `json:"displayName"`
	Rating          float64        `json:"rating"`
	UserRatingCount int            `json:"userRatingCount"`
	Reviews         []GoogleReview `json:"reviews"`
	ReviewSummary   ReviewSummary  `json:"reviewSummary"`
}
