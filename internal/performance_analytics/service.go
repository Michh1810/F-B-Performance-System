package performance_analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"fbperformance/internal/cache"
)

type Service struct {
	repo *Repository
}

const (
	performanceDashboardCacheKey = "performance_dashboard"
	dashboardSummaryCacheKey     = "dashboard_summary"
)

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetDashboardData(ctx context.Context, from, to time.Time) (SummaryDashboard, error) {
	if redisClient := cache.GetClient(); redisClient != nil {
		cached, err := redisClient.Get(ctx, dashboardSummaryCacheKey).Result()
		log.Printf("Redis GET key=%s err=%v len=%d", dashboardSummaryCacheKey, err, len(cached))
		if err == nil {
			var response SummaryDashboard
			if unmarshalErr := json.Unmarshal([]byte(cached), &response); unmarshalErr == nil {
				log.Printf("Redis cache hit: %s", dashboardSummaryCacheKey)
				return response, nil
			}
		} else {
			log.Printf("Redis cache miss: %s", dashboardSummaryCacheKey)
		}
	} else {
		log.Printf("Redis unavailable, falling back to database")
	}

	summary, err := s.repo.GetSummary(ctx, from, to)
	if err != nil {
		return SummaryDashboard{}, err
	}

	response := SummaryDashboard{
		DateRange: DateRangeConfig{
			StartDate: from,
			EndDate:   to.Add(-time.Nanosecond),
		},
		TotalRevenue:        summary.totalRevenue,
		AverageRating:       summary.averageRating,
		AverageProfitMargin: summary.averageProfitMargin,
		TotalReviews:        int(summary.totalReviews),
	}

	if redisClient := cache.GetClient(); redisClient != nil {
		payload, marshalErr := json.Marshal(response)
		if marshalErr == nil {
			err := redisClient.Set(ctx, dashboardSummaryCacheKey, payload, 5*time.Minute).Err()
			log.Printf("Redis SET key=%s err=%v", dashboardSummaryCacheKey, err)

			exists, existsErr := redisClient.Exists(ctx, dashboardSummaryCacheKey).Result()
			log.Printf("Redis EXISTS key=%s exists=%d err=%v", dashboardSummaryCacheKey, exists, existsErr)

			ttl, ttlErr := redisClient.TTL(ctx, dashboardSummaryCacheKey).Result()
			log.Printf("Redis TTL key=%s ttl=%v err=%v", dashboardSummaryCacheKey, ttl, ttlErr)

			value, verifyErr := redisClient.Get(ctx, dashboardSummaryCacheKey).Result()
			log.Printf("Redis VERIFY GET key=%s err=%v len=%d", dashboardSummaryCacheKey, verifyErr, len(value))
		}
	}

	return response, nil
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

// GetCloverOrders pulls Clover orders from Clover's API for a date range.
func (s *Service) GetCloverOrders(from, to time.Time) (*CloverOrderResponse, error) {
	merchantID := os.Getenv("CLOVER_MERCHANT_MID")
	apiToken := os.Getenv("CLOVER_DEV_API_KEY")
	baseURL := "https://apisandbox.dev.clover.com/v3/merchants/"

	fromMs := from.UnixMilli()
	toMs := to.UnixMilli()

	fullURL := fmt.Sprintf("%s%s/orders?filter=createdTime>=%d&filter=createdTime<=%d&expand=lineItems,totals&limit=100", baseURL, merchantID, fromMs, toMs)

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

// CalculateRevenueByClass computes the total revenue for each order type (Dine-in, Takeout, etc.)
func CalculateRevenueByClass(cloverData *CloverOrderResponse) map[string]float64 {
	// 1. Create a map to hold our running totals
	revenueByClassMap := make(map[string]float64)

	if cloverData == nil {
		return revenueByClassMap
	}

	// 2. Loop through every order pulled from Clover
	for _, order := range cloverData.Elements {

		// Safety check: Skip orders that weren't paid for (allow OPEN for sandbox)
		if order.PaymentState != "PAID" && order.PaymentState != "OPEN" {
			continue
		}

		className := order.OrderType.Name

		// Fallback if the order type is missing
		if className == "" {
			className = "Uncategorized"
		}

		// Clover totals are in cents (e.g., 6500 = $65.00).
		// We convert it to a float64 dollar amount for the frontend.
		dollarAmount := float64(order.Total) / 100.0

		// Add the money to the correct bucket
		revenueByClassMap[className] += dollarAmount
	}

	return revenueByClassMap
}

// GetPerformanceDashboard orchestrates Clover and Postgres data for the dashboard
func (s *Service) GetPerformanceDashboard(ctx context.Context, from, to time.Time) (*PerformanceDashboardResponse, error) {
	if redisClient := cache.GetClient(); redisClient != nil {
		cached, err := redisClient.Get(ctx, performanceDashboardCacheKey).Result()
		log.Printf("Redis GET key=%s err=%v len=%d", performanceDashboardCacheKey, err, len(cached))
		if err == nil {
			var response PerformanceDashboardResponse
			if unmarshalErr := json.Unmarshal([]byte(cached), &response); unmarshalErr == nil {
				log.Printf("Redis cache hit: %s", performanceDashboardCacheKey)
				return &response, nil
			}
		} else {
			log.Printf("Redis cache miss: %s", performanceDashboardCacheKey)
		}
	} else {
		log.Printf("Redis unavailable, falling back to database")
	}

	// Calculate Previous Period Dates
	window := to.Sub(from)
	previousTo := from
	previousFrom := previousTo.Add(-window)

	// 1. Fetch Top Items from Repo (for Master Table, Category Dominance, and Net Sales)
	rows, err := s.repo.GetTopItems(ctx, from, to, previousFrom, previousTo, "revenue", 5000)
	if err != nil {
		return nil, fmt.Errorf("failed to get top items: %w", err)
	}

	// 2. Calculate Macro KPIs from Postgres Data
	var netSales, prevNetSales float64
	for _, row := range rows {
		netSales += row.Revenue
		prevNetSales += row.Revenue / (1 + (row.TrendPercent / 100.0)) // reverse engineer previous revenue safely
	}
	
	orderCount, err := s.repo.GetOrderCount(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get order count: %w", err)
	}
	
	prevOrderCount, err := s.repo.GetOrderCount(ctx, previousFrom, previousTo)
	if err != nil {
		return nil, fmt.Errorf("failed to get previous order count: %w", err)
	}

	var avgTicketSize, prevAvgTicketSize float64
	if orderCount > 0 {
		avgTicketSize = netSales / float64(orderCount)
	}
	if prevOrderCount > 0 {
		prevAvgTicketSize = prevNetSales / float64(prevOrderCount)
	}

	calculateTrend := func(current, previous float64) float64 {
		if previous == 0 {
			return 0
		}
		return ((current - previous) / previous) * 100
	}

	netSalesTrend := calculateTrend(netSales, prevNetSales)
	ordersTrend := calculateTrend(float64(orderCount), float64(prevOrderCount))
	avgTicketSizeTrend := calculateTrend(avgTicketSize, prevAvgTicketSize)

	// 3. Calculate Revenue Classes
	revenueByClassMap, err := s.repo.GetRevenueByClass(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue by class: %w", err)
	}
	revenueClasses := make([]RevenueClassData, 0)
	for class, revenue := range revenueByClassMap {
		revenueClasses = append(revenueClasses, RevenueClassData{
			Channel: class,
			Revenue: revenue,
		})
	}

	// 5. Calculate Category Dominance
	categoryRevenueMap := make(map[string]float64)
	var topCategory string
	var maxCatRevenue float64
	totalDBRevenue := 0.0
	for _, row := range rows {
		categoryRevenueMap[row.MenuCategory] += row.Revenue
		totalDBRevenue += row.Revenue
		if categoryRevenueMap[row.MenuCategory] > maxCatRevenue {
			maxCatRevenue = categoryRevenueMap[row.MenuCategory]
			topCategory = row.MenuCategory
		}
	}

	var categoryDominancePercentage float64
	if totalDBRevenue > 0 {
		categoryDominancePercentage = (maxCatRevenue / totalDBRevenue) * 100
	}

	// 6. Build Master Table
	masterTable := make([]MasterTableItem, 0)
	for _, row := range rows {
		price := 0.0
		if row.UnitsSold > 0 {
			price = row.Revenue / float64(row.UnitsSold)
		}
		masterTable = append(masterTable, MasterTableItem{
			ID:            row.ID,
			Name:          row.Name,
			Category:      row.MenuCategory,
			Price:         price,
			UnitsSold:     row.UnitsSold,
			NetRevenue:    row.Revenue,
			GuestMentions: 0, // Mock for now
			SaleTrend:     row.TrendPercent,
		})
	}

	// Return the aggregated data
	response := &PerformanceDashboardResponse{
		KPIs: MacroKPIs{
			NetSales:          KPIMetric{Value: netSales, Trend: netSalesTrend},
			OrderTraffic:      KPIMetric{Value: float64(orderCount), Trend: ordersTrend},
			AverageTicketSize: KPIMetric{Value: avgTicketSize, Trend: avgTicketSizeTrend},
			CategoryDominance: CategoryDominance{
				CategoryName: topCategory,
				Percentage:   categoryDominancePercentage,
			},
		},
		RevenueClasses: revenueClasses,
		MasterTable:    masterTable,
	}

	if redisClient := cache.GetClient(); redisClient != nil {
		payload, marshalErr := json.Marshal(response)
		if marshalErr == nil {
			err := redisClient.Set(ctx, performanceDashboardCacheKey, payload, 5*time.Minute).Err()
			log.Printf("Redis SET key=%s err=%v", performanceDashboardCacheKey, err)

			exists, existsErr := redisClient.Exists(ctx, performanceDashboardCacheKey).Result()
			log.Printf("Redis EXISTS key=%s exists=%d err=%v", performanceDashboardCacheKey, exists, existsErr)

			ttl, ttlErr := redisClient.TTL(ctx, performanceDashboardCacheKey).Result()
			log.Printf("Redis TTL key=%s ttl=%v err=%v", performanceDashboardCacheKey, ttl, ttlErr)

			value, verifyErr := redisClient.Get(ctx, performanceDashboardCacheKey).Result()
			log.Printf("Redis VERIFY GET key=%s err=%v len=%d", performanceDashboardCacheKey, verifyErr, len(value))
		}
	}

	return response, nil
}
