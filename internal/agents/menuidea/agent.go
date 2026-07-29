// Package menuidea implements the Menu Idea Agent: given an existing menu
// item, it embeds the item as a semantic query, searches the trend-signal
// corpus for meaningfully related content, and either proposes new menu
// items (via the LLM) or surfaces a call to feature an existing item that's
// trending as-is (deterministically, no LLM needed). It is designed to run
// from a batch job (cmd/menu-idea-gen), not a live request.
//
// Splitting "this describes the existing item" from "this is a related but
// distinct idea" is done by checking whether the item's name appears in the
// caption (see mentionsItem), not by a second distance band on top of
// SignalSearcher.SearchRelevant. An earlier version tried a distance band
// and found real embeddings don't support the distinction — see
// relevanceCutoff's doc comment in internal/store/trendsignals.go.
package menuidea

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"fbperformance/internal/services/llm"
)

const (
	// signalSearchLimit bounds how many relevant signals a single
	// GenerateIdeas call considers.
	signalSearchLimit = 15

	embedTimeout    = 5 * time.Second
	generateTimeout = 15 * time.Second
)

type Agent struct {
	client       Generator
	embedder     Embedder
	signals      SignalSearcher
	model        string
	embedModel   string
	lookbackDays int
}

func NewAgent(client Generator, embedder Embedder, signals SignalSearcher, model, embedModel string, lookbackDays int) *Agent {
	return &Agent{
		client:       client,
		embedder:     embedder,
		signals:      signals,
		model:        model,
		embedModel:   embedModel,
		lookbackDays: lookbackDays,
	}
}

// ideaResponse is the JSON shape requested from the LLM.
type ideaResponse struct {
	Ideas []struct {
		Name              string `json:"name"`
		Description       string `json:"description"`
		SuggestedCategory string `json:"suggested_category"`
		Rationale         string `json:"rationale"`
	} `json:"ideas"`
}

// GenerateIdeas embeds item's name+category as a retrieval query, searches
// the trend-signal corpus for a single pool of meaningfully related
// evidence (SearchRelevant), then splits that pool by caption text:
//   - signals whose caption names the item become a "promotion" idea,
//     synthesized deterministically with no LLM call — the item already
//     exists, there's nothing to invent, just evidence it's trending as-is.
//   - the remaining signals (related but not naming the item), if there's
//     enough evidence, are sent to the LLM for 0-2 concrete "tweak" ideas —
//     new items distinct from item itself.
//
// Mirrors Trend Agent's graceful-degradation convention: an embed/search
// failure or zero relevant signals are not errors, they just mean nothing
// gets proposed this run. Only the LLM generation call itself is a hard
// failure, left for the caller (the batch loop) to log per-item and
// continue with the next one — any promotion idea already found is still
// returned alongside that error.
//
// The returned CallLog records what actually happened on every path — how
// many signals fell into each bucket, which subset reached the LLM, the
// exact prompt and raw response — so callers can persist it for debugging
// even when GenerateIdeas degrades gracefully or hard-fails.
func (a *Agent) GenerateIdeas(ctx context.Context, item MenuItem) ([]IdeaCandidate, CallLog, error) {
	callLog := CallLog{
		MenuItemID:   item.ID,
		MenuItemName: item.Name,
		CalledAt:     time.Now().UTC(),
	}

	queryEmbedding, err := a.embedQuery(ctx, fmt.Sprintf("%s (%s)", item.Name, item.Category))
	if err != nil {
		callLog.Error = fmt.Sprintf("embed query: %v", err)
		return nil, callLog, nil
	}

	since := time.Now().AddDate(0, 0, -a.lookbackDays)
	relevant, err := a.signals.SearchRelevant(ctx, queryEmbedding, signalSearchLimit, since)
	if err != nil {
		callLog.Error = fmt.Sprintf("search relevant: %v", err)
		return nil, callLog, nil
	}
	if len(relevant) == 0 {
		return nil, callLog, nil
	}

	promotionSignals, tweakSignals := splitByMention(relevant, item.Name)
	callLog.PromotionSignalCount = len(promotionSignals)
	callLog.AdjacentSignalCount = len(tweakSignals)

	var candidates []IdeaCandidate
	if len(promotionSignals) > 0 {
		candidates = append(candidates, buildPromotionCandidate(item, selectTopAdjacentSignals(promotionSignals, maxPromptSignals)))
	}

	if len(tweakSignals) == 0 {
		callLog.IdeaCount = len(candidates)
		return candidates, callLog, nil
	}

	promptSignals := selectTopAdjacentSignals(tweakSignals, maxPromptSignals)
	callLog.PromptSignals = promptSignalRefs(promptSignals)
	callLog.PromptText = buildIdeaPrompt(item, tweakSignals)

	generateCtx, cancel := context.WithTimeout(ctx, generateTimeout)
	text, err := a.client.Generate(generateCtx, a.model, callLog.PromptText, llm.GenerateOptions{
		SystemPrompt: systemPrompt,
		JSONMode:     true,
	})
	cancel()
	callLog.RawResponse = text
	if err != nil {
		callLog.Error = fmt.Sprintf("generate ideas: %v", err)
		callLog.IdeaCount = len(candidates)
		return candidates, callLog, fmt.Errorf("menuidea agent: generate ideas: %w", err)
	}

	var resp ideaResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		callLog.Error = fmt.Sprintf("parse idea response: %v", err)
		callLog.IdeaCount = len(candidates)
		return candidates, callLog, fmt.Errorf("menuidea agent: parse idea response: %w", err)
	}

	hashtags, videoURLs, totalViews := summarizeSignals(promptSignals)
	for _, idea := range resp.Ideas {
		candidates = append(candidates, IdeaCandidate{
			Kind:              "tweak",
			Name:              idea.Name,
			Description:       idea.Description,
			SuggestedCategory: idea.SuggestedCategory,
			Rationale:         idea.Rationale,
			SourceHashtags:    hashtags,
			SourceSignalCount: len(promptSignals),
			SourceTotalViews:  totalViews,
			SourceVideoURLs:   videoURLs,
		})
	}
	callLog.IdeaCount = len(candidates)
	return candidates, callLog, nil
}

// buildPromotionCandidate synthesizes a "feature this existing item" idea
// directly from signals whose caption names the item — no LLM call, since
// the item already exists and there's nothing to invent, just evidence
// it's currently trending as-is.
func buildPromotionCandidate(item MenuItem, signals []RelevantSignal) IdeaCandidate {
	hashtags, videoURLs, totalViews := summarizeSignals(signals)
	return IdeaCandidate{
		Kind:              "promotion",
		Name:              item.Name,
		Description:       fmt.Sprintf("Feature %s now — TikTok content mentioning it by name is currently trending.", item.Name),
		SuggestedCategory: item.Category,
		Rationale: fmt.Sprintf(
			"%d TikTok video(s) naming %s directly were found in the trend corpus, totaling %d views.",
			len(signals), item.Name, totalViews,
		),
		SourceHashtags:    hashtags,
		SourceSignalCount: len(signals),
		SourceTotalViews:  totalViews,
		SourceVideoURLs:   videoURLs,
	}
}

// splitByMention partitions relevant signals into promotion evidence
// (caption names the item) and tweak-candidate evidence (it doesn't) — see
// mentionsItem.
func splitByMention(signals []RelevantSignal, itemName string) (promotion, tweak []RelevantSignal) {
	for _, s := range signals {
		if mentionsItem(s.Caption, itemName) {
			promotion = append(promotion, s)
		} else {
			tweak = append(tweak, s)
		}
	}
	return promotion, tweak
}

// mentionsItem reports whether caption names itemName (case-insensitive
// substring match) — a cheap, deterministic stand-in for "this describes
// the existing item" that distance thresholding can't reliably provide
// (see relevanceCutoff's doc comment in internal/store/trendsignals.go).
func mentionsItem(caption, itemName string) bool {
	return strings.Contains(strings.ToLower(caption), strings.ToLower(itemName))
}

// promptSignalRefs extracts the identifying fields of each signal actually
// shown to the LLM, for CallLog.
func promptSignalRefs(signals []RelevantSignal) []PromptSignalRef {
	refs := make([]PromptSignalRef, len(signals))
	for i, s := range signals {
		refs[i] = PromptSignalRef{
			ID:         s.ID,
			ExternalID: s.ExternalID,
			Caption:    s.Caption,
			ViewCount:  s.ViewCount,
			URL:        s.URL,
		}
	}
	return refs
}

// embedQuery embeds text as a search query (not a document — see
// llm.EmbedOptions.TaskType) for semantic retrieval against the corpus.
func (a *Agent) embedQuery(ctx context.Context, text string) ([]float32, error) {
	embedCtx, cancel := context.WithTimeout(ctx, embedTimeout)
	defer cancel()
	return a.embedder.Embed(embedCtx, a.embedModel, text, llm.EmbedOptions{
		TaskType:             "RETRIEVAL_QUERY",
		OutputDimensionality: EmbeddingDimensions,
	})
}

// summarizeSignals computes the evidence attached to every idea candidate
// from a single GenerateIdeas call: the unique hashtags and video URLs
// across the signals actually shown to the LLM (first-seen order, capped),
// and their combined view count.
func summarizeSignals(signals []RelevantSignal) (hashtags []string, videoURLs []string, totalViews int64) {
	seenHashtags := make(map[string]bool)
	seenURLs := make(map[string]bool)
	for _, s := range signals {
		totalViews += s.ViewCount
		for _, h := range s.Hashtags {
			if h == "" || seenHashtags[h] {
				continue
			}
			seenHashtags[h] = true
			hashtags = append(hashtags, h)
			if len(hashtags) >= 10 {
				break
			}
		}
		if s.URL == "" || seenURLs[s.URL] {
			continue
		}
		seenURLs[s.URL] = true
		videoURLs = append(videoURLs, s.URL)
	}
	return hashtags, videoURLs, totalViews
}
