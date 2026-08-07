package hashtagsuggest

import (
	"context"

	"fbperformance/internal/services/llm"
)

// Generator is the subset of the LLM client this package depends on to
// suggest hashtags from a restaurant description.
type Generator interface {
	Generate(ctx context.Context, model, prompt string, opts llm.GenerateOptions) (string, error)
}

// MenuItemRef is the minimal menu item context used to ground hashtag
// suggestions in what the restaurant actually sells, kept local rather
// than importing menuidea.MenuItem so this package doesn't depend on
// another agent package for a two-field DTO.
type MenuItemRef struct {
	Name     string
	Category string
}

// Input is what Suggest needs: the restaurant's free-text description plus
// its current active menu, for grounding.
type Input struct {
	Description string
	MenuItems   []MenuItemRef
}
