package orchestrator

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"fbperformance/internal/agents/financial"
	"fbperformance/internal/agents/manager"
	"fbperformance/internal/agents/trend"
)

type fakeTrendAgent struct {
	calls  int
	result trend.Result
	err    error
}

func (f *fakeTrendAgent) Analyze(ctx context.Context, in trend.Input) (trend.Result, error) {
	f.calls++
	return f.result, f.err
}

type fakeFinancialAgent struct {
	calls int
	text  string
	err   error
}

func (f *fakeFinancialAgent) Analyze(ctx context.Context, in financial.Input) (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	return f.text, nil
}

type fakeManagerAgent struct {
	calls int
	in    manager.Input
	out   *manager.Output
	err   error
}

func (f *fakeManagerAgent) Decide(ctx context.Context, in manager.Input) (*manager.Output, error) {
	f.calls++
	f.in = in
	if f.err != nil {
		return nil, f.err
	}
	return f.out, nil
}

type fakeFlags struct {
	calls   int
	enabled bool
	err     error
}

func (f *fakeFlags) FinancialAgentEnabled(ctx context.Context) (bool, error) {
	f.calls++
	return f.enabled, f.err
}

func baseTrendResult() trend.Result {
	return trend.Result{
		Summary:           "trend summary",
		VideoCount:        8,
		TotalViews:        1200,
		TotalLikes:        120,
		TotalComments:     12,
		TotalShares:       6,
		EngagementRate:    0.115,
		SentimentLabel:    "positive",
		SentimentScore:    0.8,
		GrowthRatePct:     nil,
		GrowthPeriodHours: 0,
		TopHashtags:       []string{"food"},
		RelatedTrends:     []string{"spicy"},
	}
}

func TestGetRecommendation_FlagOn_RunsFinancialAndTrend(t *testing.T) {
	trendAgent := &fakeTrendAgent{result: baseTrendResult()}
	financialAgent := &fakeFinancialAgent{text: "financial summary"}
	managerAgent := &fakeManagerAgent{out: &manager.Output{Decision: manager.DecisionLaunch, Reasoning: "go"}}
	flags := &fakeFlags{enabled: true}
	o := New(trendAgent, financialAgent, managerAgent, flags)

	resp, err := o.GetRecommendation(context.Background(), Request{MenuItemID: uuid.New(), ItemName: "Bibimbap"})
	if err != nil {
		t.Fatalf("GetRecommendation() error = %v, want nil", err)
	}
	if trendAgent.calls != 1 {
		t.Fatalf("trend calls = %d, want 1", trendAgent.calls)
	}
	if financialAgent.calls != 1 {
		t.Fatalf("financial calls = %d, want 1", financialAgent.calls)
	}
	if managerAgent.calls != 1 {
		t.Fatalf("manager calls = %d, want 1", managerAgent.calls)
	}
	if resp.FinancialAnalysis != "financial summary" {
		t.Fatalf("FinancialAnalysis = %q, want %q", resp.FinancialAnalysis, "financial summary")
	}
}

func TestGetRecommendation_FlagOff_SkipsFinancialButRunsTrend(t *testing.T) {
	trendAgent := &fakeTrendAgent{result: baseTrendResult()}
	financialAgent := &fakeFinancialAgent{text: "should-not-run"}
	managerAgent := &fakeManagerAgent{out: &manager.Output{Decision: manager.DecisionReprice, Reasoning: "trend only"}}
	flags := &fakeFlags{enabled: false}
	o := New(trendAgent, financialAgent, managerAgent, flags)

	resp, err := o.GetRecommendation(context.Background(), Request{MenuItemID: uuid.New(), ItemName: "Bibimbap"})
	if err != nil {
		t.Fatalf("GetRecommendation() error = %v, want nil", err)
	}
	if trendAgent.calls != 1 {
		t.Fatalf("trend calls = %d, want 1", trendAgent.calls)
	}
	if financialAgent.calls != 0 {
		t.Fatalf("financial calls = %d, want 0", financialAgent.calls)
	}
	if managerAgent.calls != 1 {
		t.Fatalf("manager calls = %d, want 1", managerAgent.calls)
	}
	if resp.FinancialAnalysis != financialFlagOffFallback {
		t.Fatalf("FinancialAnalysis = %q, want %q", resp.FinancialAnalysis, financialFlagOffFallback)
	}
	if managerAgent.in.FinancialAnalysis != financialFlagOffFallback {
		t.Fatalf("manager FinancialAnalysis = %q, want %q", managerAgent.in.FinancialAnalysis, financialFlagOffFallback)
	}
}

func TestGetRecommendation_FlagEvaluationError_FailClosed(t *testing.T) {
	trendAgent := &fakeTrendAgent{result: baseTrendResult()}
	financialAgent := &fakeFinancialAgent{text: "should-not-run"}
	managerAgent := &fakeManagerAgent{out: &manager.Output{Decision: manager.DecisionCut, Reasoning: "safe mode"}}
	flags := &fakeFlags{err: errors.New("posthog timeout")}
	o := New(trendAgent, financialAgent, managerAgent, flags)

	resp, err := o.GetRecommendation(context.Background(), Request{MenuItemID: uuid.New(), ItemName: "Bibimbap"})
	if err != nil {
		t.Fatalf("GetRecommendation() error = %v, want nil", err)
	}
	if trendAgent.calls != 1 {
		t.Fatalf("trend calls = %d, want 1", trendAgent.calls)
	}
	if financialAgent.calls != 0 {
		t.Fatalf("financial calls = %d, want 0", financialAgent.calls)
	}
	if managerAgent.calls != 1 {
		t.Fatalf("manager calls = %d, want 1", managerAgent.calls)
	}
	if resp.FinancialAnalysis != financialFlagErrorFallback {
		t.Fatalf("FinancialAnalysis = %q, want %q", resp.FinancialAnalysis, financialFlagErrorFallback)
	}
}

func TestGetRecommendation_FlagOn_FinancialErrorUnchanged(t *testing.T) {
	trendAgent := &fakeTrendAgent{result: baseTrendResult()}
	financialAgent := &fakeFinancialAgent{err: errors.New("forecast failed")}
	managerAgent := &fakeManagerAgent{out: &manager.Output{Decision: manager.DecisionLaunch, Reasoning: "unused"}}
	flags := &fakeFlags{enabled: true}
	o := New(trendAgent, financialAgent, managerAgent, flags)

	_, err := o.GetRecommendation(context.Background(), Request{MenuItemID: uuid.New(), ItemName: "Bibimbap"})
	if err == nil {
		t.Fatalf("expected error when financial agent fails and flag is ON")
	}
	if managerAgent.calls != 0 {
		t.Fatalf("manager calls = %d, want 0 when financial fails", managerAgent.calls)
	}
}

func TestGetRecommendation_FlagOff_TrendErrorStillFails(t *testing.T) {
	trendAgent := &fakeTrendAgent{err: errors.New("trend failure")}
	financialAgent := &fakeFinancialAgent{text: "should-not-run"}
	managerAgent := &fakeManagerAgent{out: &manager.Output{Decision: manager.DecisionLaunch, Reasoning: "unused"}}
	flags := &fakeFlags{enabled: false}
	o := New(trendAgent, financialAgent, managerAgent, flags)

	_, err := o.GetRecommendation(context.Background(), Request{MenuItemID: uuid.New(), ItemName: "Bibimbap"})
	if err == nil {
		t.Fatalf("expected trend error to propagate")
	}
	if financialAgent.calls != 0 {
		t.Fatalf("financial calls = %d, want 0", financialAgent.calls)
	}
	if managerAgent.calls != 0 {
		t.Fatalf("manager calls = %d, want 0 when trend fails", managerAgent.calls)
	}
}

func TestGetRecommendation_FlagOff_DoesNotInvokeFinancial(t *testing.T) {
	trendAgent := &fakeTrendAgent{result: baseTrendResult()}
	financialAgent := &fakeFinancialAgent{err: errors.New("must not be called")}
	managerAgent := &fakeManagerAgent{out: &manager.Output{Decision: manager.DecisionLaunch, Reasoning: "ok"}}
	flags := &fakeFlags{enabled: false}
	o := New(trendAgent, financialAgent, managerAgent, flags)

	_, err := o.GetRecommendation(context.Background(), Request{MenuItemID: uuid.New(), ItemName: "Bibimbap"})
	if err != nil {
		t.Fatalf("GetRecommendation() error = %v, want nil", err)
	}
	if financialAgent.calls != 0 {
		t.Fatalf("financial calls = %d, want 0", financialAgent.calls)
	}
}
