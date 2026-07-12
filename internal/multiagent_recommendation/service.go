package multiagent_recommendation

import (
	"context"

	"fbperformance/internal/gemini"
)

type Service struct {
	gemini *gemini.Client
	model  string
}

func NewService(client *gemini.Client, model string) *Service {
	return &Service{gemini: client, model: model}
}

func (s *Service) GetRecommendation(ctx context.Context, prompt string) (string, error) {
	return s.gemini.GenerateContent(ctx, s.model, prompt)
}
