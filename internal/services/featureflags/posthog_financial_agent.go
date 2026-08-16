package featureflags

import (
	"context"
	"fmt"
	"time"

	"github.com/posthog/posthog-go"
)

const defaultEvaluationTimeout = 200 * time.Millisecond

type PostHogFinancialAgent struct {
	client     posthog.Client
	distinctID string
	timeout    time.Duration
	initErr    error
}

func NewPostHogFinancialAgent(apiKey, endpoint, distinctID string, timeout time.Duration) *PostHogFinancialAgent {
	if distinctID == "" {
		distinctID = defaultDistinctID
	}
	if timeout <= 0 {
		timeout = defaultEvaluationTimeout
	}

	cfg := posthog.Config{
		FeatureFlagRequestTimeout:    timeout,
		FeatureFlagRequestMaxRetries: posthog.Ptr(0),
	}
	if endpoint != "" {
		cfg.Endpoint = endpoint
	}

	client, err := posthog.NewWithConfig(apiKey, cfg)
	if err != nil {
		return &PostHogFinancialAgent{
			distinctID: distinctID,
			timeout:    timeout,
			initErr:    fmt.Errorf("featureflags: init posthog client: %w", err),
		}
	}

	return &PostHogFinancialAgent{
		client:     client,
		distinctID: distinctID,
		timeout:    timeout,
	}
}

func (p *PostHogFinancialAgent) Close() error {
	if p.client == nil {
		return nil
	}
	return p.client.Close()
}

func (p *PostHogFinancialAgent) FinancialAgentEnabled(ctx context.Context) (bool, error) {
	if p.initErr != nil {
		return false, p.initErr
	}
	if p.client == nil {
		return false, fmt.Errorf("featureflags: posthog client is not initialized")
	}

	evalCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	evals, err := posthog.EvaluateFlagsWithContext(evalCtx, p.client, posthog.EvaluateFlagsPayload{
		DistinctId: p.distinctID,
		FlagKeys:   []string{FinancialAgentEnabledKey},
	})
	if err != nil {
		return false, fmt.Errorf("featureflags: evaluate %q: %w", FinancialAgentEnabledKey, err)
	}
	if evals == nil {
		return false, fmt.Errorf("featureflags: evaluate %q: empty response", FinancialAgentEnabledKey)
	}

	return evals.IsEnabled(FinancialAgentEnabledKey), nil
}
