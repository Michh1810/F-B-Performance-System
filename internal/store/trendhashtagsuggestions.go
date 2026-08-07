package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrHashtagSuggestionNotFound is returned by UpdateStatus when id doesn't
// match any row.
var ErrHashtagSuggestionNotFound = errors.New("trend hashtag suggestion not found")

// HashtagSuggestion is one hashtagsuggest.Agent.Suggest call's output,
// pending human review before cmd/trend-ingest will sweep it.
type HashtagSuggestion struct {
	ID                uuid.UUID  `json:"id"`
	SourceDescription string     `json:"source_description"`
	Hashtags          []string   `json:"hashtags"`
	Status            string     `json:"status"`
	GeneratedAt       time.Time  `json:"generated_at"`
	ReviewedAt        *time.Time `json:"reviewed_at"`
}

// TrendHashtagSuggestionStore persists and retrieves hashtag suggestions
// for human review.
type TrendHashtagSuggestionStore struct {
	pool *pgxpool.Pool
}

func NewTrendHashtagSuggestionStore(pool *pgxpool.Pool) *TrendHashtagSuggestionStore {
	return &TrendHashtagSuggestionStore{pool: pool}
}

// Save persists a newly generated suggestion with status "pending".
func (s *TrendHashtagSuggestionStore) Save(ctx context.Context, sourceDescription string, hashtags []string) (HashtagSuggestion, error) {
	hashtagsJSON, err := json.Marshal(hashtags)
	if err != nil {
		return HashtagSuggestion{}, fmt.Errorf("store: marshal hashtags: %w", err)
	}

	suggestion := HashtagSuggestion{SourceDescription: sourceDescription, Hashtags: hashtags}
	err = s.pool.QueryRow(ctx, `
		INSERT INTO trend_hashtag_suggestions (source_description, hashtags)
		VALUES ($1, $2::jsonb)
		RETURNING id, status, generated_at, reviewed_at`,
		sourceDescription, string(hashtagsJSON),
	).Scan(&suggestion.ID, &suggestion.Status, &suggestion.GeneratedAt, &suggestion.ReviewedAt)
	if err != nil {
		return HashtagSuggestion{}, fmt.Errorf("store: save trend hashtag suggestion: %w", err)
	}
	return suggestion, nil
}

// List returns suggestions, most recently generated first, optionally
// filtered by status ("" means no filter).
func (s *TrendHashtagSuggestionStore) List(ctx context.Context, status string) ([]HashtagSuggestion, error) {
	query := `SELECT id, source_description, hashtags, status, generated_at, reviewed_at FROM trend_hashtag_suggestions`

	var rows pgx.Rows
	var err error
	if status != "" {
		rows, err = s.pool.Query(ctx, query+` WHERE status = $1 ORDER BY generated_at DESC`, status)
	} else {
		rows, err = s.pool.Query(ctx, query+` ORDER BY generated_at DESC`)
	}
	if err != nil {
		return nil, fmt.Errorf("store: list trend hashtag suggestions: %w", err)
	}
	defer rows.Close()

	var suggestions []HashtagSuggestion
	for rows.Next() {
		var suggestion HashtagSuggestion
		var hashtags []byte
		if err := rows.Scan(
			&suggestion.ID, &suggestion.SourceDescription, &hashtags,
			&suggestion.Status, &suggestion.GeneratedAt, &suggestion.ReviewedAt,
		); err != nil {
			return nil, fmt.Errorf("store: scan trend hashtag suggestion: %w", err)
		}
		if err := json.Unmarshal(hashtags, &suggestion.Hashtags); err != nil {
			return nil, fmt.Errorf("store: unmarshal hashtags: %w", err)
		}
		suggestions = append(suggestions, suggestion)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list trend hashtag suggestions: %w", err)
	}
	return suggestions, nil
}

// UpdateStatus transitions a suggestion's review status and stamps
// reviewed_at. Returns ErrHashtagSuggestionNotFound if id doesn't match any
// row.
func (s *TrendHashtagSuggestionStore) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE trend_hashtag_suggestions SET status = $1, reviewed_at = CURRENT_TIMESTAMP WHERE id = $2`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("store: update trend hashtag suggestion status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrHashtagSuggestionNotFound
	}
	return nil
}

// LatestApproved returns the hashtags from the most recently approved
// suggestion, or nil if none has been approved yet. This is what
// cmd/trend-ingest sweeps.
func (s *TrendHashtagSuggestionStore) LatestApproved(ctx context.Context) ([]string, error) {
	var hashtags []byte
	err := s.pool.QueryRow(ctx, `
		SELECT hashtags FROM trend_hashtag_suggestions
		WHERE status = 'approved'
		ORDER BY reviewed_at DESC
		LIMIT 1`,
	).Scan(&hashtags)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("store: get latest approved trend hashtags: %w", err)
	}

	var result []string
	if err := json.Unmarshal(hashtags, &result); err != nil {
		return nil, fmt.Errorf("store: unmarshal hashtags: %w", err)
	}
	return result, nil
}
