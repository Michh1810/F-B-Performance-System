package featureflags

import "context"

const (
	FinancialAgentEnabledKey = "financial-agent-enabled"
	defaultDistinctID        = "fbperformance-backend"
)

// FeatureFlags is the minimal surface the orchestrator needs for the
// Financial Agent kill switch.
type FeatureFlags interface {
	FinancialAgentEnabled(ctx context.Context) (bool, error)
}
