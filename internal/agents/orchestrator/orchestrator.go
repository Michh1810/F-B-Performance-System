// Package orchestrator coordinates the Trend, Financial, and Manager
// agents into a single menu-item recommendation pipeline. It owns agent
// sequencing and concurrency; it holds no prompt or provider knowledge of
// its own.
package orchestrator

import (
	"context"
	"fmt"
	"sync"

	"fbperformance/internal/agents/financial"
	"fbperformance/internal/agents/manager"
	"fbperformance/internal/agents/trend"
)

type Orchestrator struct {
	trend     *trend.Agent
	financial *financial.Agent
	manager   *manager.Agent
}

func New(trendAgent *trend.Agent, financialAgent *financial.Agent, managerAgent *manager.Agent) *Orchestrator {
	return &Orchestrator{trend: trendAgent, financial: financialAgent, manager: managerAgent}
}

// financialOutcome carries the Financial Agent's result across its goroutine.
type financialOutcome struct {
	text string
	err  error
}

// trendOutcome carries the Trend Agent's result across its goroutine.
type trendOutcome struct {
	result trend.Result
	err    error
}

// GetRecommendation runs the Trend Agent and Financial Agent concurrently,
// waits for both to finish, then feeds their analyses to the Manager Agent
// for a final structured decision.
func (o *Orchestrator) GetRecommendation(ctx context.Context, req Request) (*Response, error) {
	var wg sync.WaitGroup
	var trendOut trendOutcome
	var financialOut financialOutcome

	wg.Add(2)
	go func() {
		defer wg.Done()
		trendOut.result, trendOut.err = o.trend.Analyze(ctx, trend.Input{
			MenuItemID: req.MenuItemID,
			ItemName:   req.ItemName,
		})
	}()
	go func() {
		defer wg.Done()
		financialOut.text, financialOut.err = o.financial.Analyze(ctx, financial.Input{
			ItemName:      req.ItemName,
			FinancialData: req.FinancialData,
		})
	}()
	wg.Wait()

	if trendOut.err != nil {
		return nil, trendOut.err
	}
	if financialOut.err != nil {
		return nil, financialOut.err
	}

	decision, err := o.manager.Decide(ctx, manager.Input{
		ItemName:          req.ItemName,
		TrendAnalysis:     trendOut.result.Summary,
		FinancialAnalysis: financialOut.text,

		TrendVideoCount:        trendOut.result.VideoCount,
		TrendTotalViews:        trendOut.result.TotalViews,
		TrendEngagementRate:    trendOut.result.EngagementRate,
		TrendGrowthRatePct:     trendOut.result.GrowthRatePct,
		TrendGrowthPeriodHours: trendOut.result.GrowthPeriodHours,
		TrendSentimentLabel:    trendOut.result.SentimentLabel,
		TrendSentimentScore:    trendOut.result.SentimentScore,
		TrendRelatedFoodTrends: trendOut.result.RelatedTrends,
	})
	if err != nil {
		return nil, fmt.Errorf("orchestrator: %w", err)
	}

	return &Response{
		ItemName:          req.ItemName,
		Decision:          decision.Decision,
		Reasoning:         decision.Reasoning,
		TrendAnalysis:     trendOut.result.Summary,
		FinancialAnalysis: financialOut.text,
		TrendMetrics: TrendMetrics{
			VideoCount:        trendOut.result.VideoCount,
			TotalViews:        trendOut.result.TotalViews,
			TotalLikes:        trendOut.result.TotalLikes,
			TotalComments:     trendOut.result.TotalComments,
			TotalShares:       trendOut.result.TotalShares,
			EngagementRate:    trendOut.result.EngagementRate,
			SentimentLabel:    trendOut.result.SentimentLabel,
			SentimentScore:    trendOut.result.SentimentScore,
			GrowthRatePct:     trendOut.result.GrowthRatePct,
			GrowthPeriodHours: trendOut.result.GrowthPeriodHours,
			TopHashtags:       trendOut.result.TopHashtags,
			RelatedTrends:     trendOut.result.RelatedTrends,
		},
	}, nil
}
