package manager

import (
	"context"

	"fbperformance/internal/services/llm"
)

// Generator is the subset of the LLM client the Manager Agent depends on.
type Generator interface {
	Generate(ctx context.Context, model, prompt string, opts llm.GenerateOptions) (string, error)
}

// Decision is the Manager Agent's final call on a menu item.
type Decision string

const (
	DecisionLaunch  Decision = "LAUNCH"
	DecisionCut     Decision = "CUT"
	DecisionReprice Decision = "REPRICE"
)

// Input is the synthesized context the Manager Agent decides over: the
// upstream Trend Agent and Financial Agent analyses for one menu item.
// Trend metrics are passed as flattened primitive fields (not a trend.Result
// import) so the Manager Agent stays decoupled from the other agent
// packages, matching its existing zero-cross-agent-import style.
type Input struct {
	ItemName          string
	TrendAnalysis     string
	FinancialAnalysis string

	TrendVideoCount        int
	TrendTotalViews        int64
	TrendEngagementRate    float64
	TrendGrowthRatePct     *float64
	TrendGrowthPeriodHours float64
	TrendSentimentLabel    string
	TrendSentimentScore    float64
	TrendRelatedFoodTrends []string
}

// Output is the structured JSON shape the Manager Agent is prompted to
// return.
type Output struct {
	Decision  Decision `json:"decision"`
	Reasoning string   `json:"reasoning"`
}
