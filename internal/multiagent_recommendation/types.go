package multiagent_recommendation

// RecommendationRequest is the client-supplied prompt asking the AI for a recommendation.
type RecommendationRequest struct {
	Prompt string `json:"prompt"`
}

// RecommendationResponse wraps the AI-generated recommendation text.
type RecommendationResponse struct {
	Recommendation string `json:"recommendation"`
}
