package cache

import (
	"context"
	"log"
)

const (
	KeyOverview             = "overview"
	KeyPerformanceDashboard = "performance_dashboard"
	KeyDashboardSummary     = "dashboard_summary"
)

func InvalidateOverview(ctx context.Context) {
	deleteKey(ctx, KeyOverview)
}

func InvalidatePerformanceDashboard(ctx context.Context) {
	deleteKey(ctx, KeyPerformanceDashboard)
}

func InvalidateDashboardSummary(ctx context.Context) {
	deleteKey(ctx, KeyDashboardSummary)
}

func InvalidateAnalyticsCache(ctx context.Context) {
	if client := GetClient(); client != nil {
		_ = client.Del(ctx, KeyOverview, KeyPerformanceDashboard, KeyDashboardSummary).Err()
	}
}

func deleteKey(ctx context.Context, key string) {
	if client := GetClient(); client != nil {
		if err := client.Del(ctx, key).Err(); err == nil {
			log.Printf("Redis cache invalidated: %s", key)
		}
	}
}
