package overview

type OverviewResponse struct {
	CriticalAlert *CriticalAlert `json:"criticalAlert,omitempty"`
	WeeklyPulse   WeeklyPulse    `json:"weeklyPulse"`
	AIInsights    []AIInsight    `json:"aiInsights"`
}

type CriticalAlert struct {
	Active  bool   `json:"active"`
	Type    string `json:"type"` // "warning" or "destructive"
	Title   string `json:"title"`
	Message string `json:"message"`
}

type WeeklyPulse struct {
	Revenue   Metric `json:"revenue"`
	AOV       Metric `json:"aov"`
	Orders    Metric `json:"orders"`
	Sentiment Metric `json:"sentiment"`
}

type Metric struct {
	Value   float64 `json:"value"`
	Label   string  `json:"label"`
	Trend   string  `json:"trend"` // e.g., "+5%"
	TrendUp bool    `json:"trendUp"`
	Vs      string  `json:"vs"` // e.g., "previous week"
}

type AIInsight struct {
	ID         string `json:"id"`
	Pillar     string `json:"pillar"`
	Title      string `json:"title"`
	Icon       string `json:"icon"` // "trendingUp", "dollarSign", "alertCircle", "messageSquare"
	Suggestion string `json:"suggestion"`
}
