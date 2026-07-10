// This is the brain of feature. It contains the actual raw Go logic, mathematical algorithms, calculations, and database coordination.
//  It calculates net profit (`revenue - cogs`), averages review ratings, and sorts the top items.
package performance_analytics

type Service struct{ // declare an empty structure that represents business tool, when connect real data base, we must update this
	//db *pgx.Pool
	// redis *redis.Client
} 

func NewService() *Service {
	return &Service{}
}

// You place (s *Service) right before the function name. 
// This is Go's version of writing def get_data(self): in Python. // This function must build and return a "SummaryDashboard" struct
func (s *Service) GetDashBoardData() SummaryDashboard{
	now := time.Now()
	start := now.AddDate(0, 0, -30)
	end := now

	data := SummaryDashboard{
		DateRange: DateRangeConfig{
			StartDate: start,
			EndDate: end,},
			//  this is mock JSON Data
			TotalRevenue: 14500.75
			AverageRating: 4.65
			AverageProfitMargin: 0.42
			TotalReviews: 129,
		
		}

	return data
		
	}
}