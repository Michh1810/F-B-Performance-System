package orchestrator

import (
	"github.com/google/uuid"

	"fbperformance/internal/agents/manager"
)

// Request is the client-supplied context for a menu item decision:
// MenuItemID identifies the item for TikTok trend lookup (see trend
// package), and FinancialData is the raw financial/sales signal the
// Financial Agent analyzes.
type Request struct {
	MenuItemID    uuid.UUID `json:"menu_item_id"`
	ItemName      string    `json:"item_name"`
	FinancialData string    `json:"financial_data"`
}

// Response is the synthesized output of the multi-agent pipeline: the
// Manager Agent's decision plus the underlying agent analyses that informed
// it, including the structured TikTok trend metrics.
type Response struct {
	ItemName          string           `json:"item_name"`
	Decision          manager.Decision `json:"decision"`
	Reasoning         string           `json:"reasoning"`
	TrendAnalysis     string           `json:"trend_analysis"`
	FinancialAnalysis string           `json:"financial_analysis"`

	TrendMetrics TrendMetrics `json:"trend_metrics"`
}

// TrendMetrics is the structured TikTok signal for the menu item, mirroring
// trend.Result's fields for API-client consumption.
type TrendMetrics struct {
	VideoCount        int      `json:"video_count"`
	TotalViews        int64    `json:"total_views"`
	TotalLikes        int64    `json:"total_likes"`
	TotalComments     int64    `json:"total_comments"`
	TotalShares       int64    `json:"total_shares"`
	EngagementRate    float64  `json:"engagement_rate"`
	SentimentLabel    string   `json:"sentiment_label"`
	SentimentScore    float64  `json:"sentiment_score"`
	GrowthRatePct     *float64 `json:"growth_rate_pct"`
	GrowthPeriodHours float64  `json:"growth_period_hours"`
	TopHashtags       []string `json:"top_hashtags"`
	RelatedTrends     []string `json:"related_trends"`
}
