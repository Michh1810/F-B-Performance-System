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
type MenuItems struct {

}

type SalesTrend struct {

}

type ReviewSummary struct {

}
