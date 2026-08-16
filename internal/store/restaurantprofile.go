package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// restaurantProfileSingletonID is the fixed row id restaurant_profile
// always uses. There is exactly one restaurant per deployment, so Upsert
// always targets this id via ON CONFLICT rather than tracking "the current
// row" separately.
var restaurantProfileSingletonID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// RestaurantProfile is this restaurant's free-text description, entered via
// the frontend and used to ground hashtagsuggest.Agent's suggestions in
// what kind of food this specific restaurant sells. UpdatedAt is nil when
// no profile has been set yet.
type RestaurantProfile struct {
	Description string     `json:"description"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

// RestaurantProfileStore persists the single restaurant_profile row.
type RestaurantProfileStore struct {
	pool *pgxpool.Pool
}

func NewRestaurantProfileStore(pool *pgxpool.Pool) *RestaurantProfileStore {
	return &RestaurantProfileStore{pool: pool}
}

// Get returns the current profile, or a zero-value RestaurantProfile
// (empty Description, nil UpdatedAt) if none has been set yet.
func (s *RestaurantProfileStore) Get(ctx context.Context) (RestaurantProfile, error) {
	var profile RestaurantProfile
	err := s.pool.QueryRow(ctx, `SELECT description, updated_at FROM restaurant_profile WHERE id = $1`, restaurantProfileSingletonID).
		Scan(&profile.Description, &profile.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return RestaurantProfile{}, nil
		}
		return RestaurantProfile{}, fmt.Errorf("store: get restaurant profile: %w", err)
	}
	return profile, nil
}

// Upsert sets the restaurant's description, creating the singleton row if
// it doesn't exist yet.
func (s *RestaurantProfileStore) Upsert(ctx context.Context, description string) (RestaurantProfile, error) {
	var profile RestaurantProfile
	err := s.pool.QueryRow(ctx, `
		INSERT INTO restaurant_profile (id, description)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET description = EXCLUDED.description, updated_at = CURRENT_TIMESTAMP
		RETURNING description, updated_at`,
		restaurantProfileSingletonID, description,
	).Scan(&profile.Description, &profile.UpdatedAt)
	if err != nil {
		return RestaurantProfile{}, fmt.Errorf("store: upsert restaurant profile: %w", err)
	}
	return profile, nil
}
