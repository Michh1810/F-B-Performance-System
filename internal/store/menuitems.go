package store

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MenuItem is the subset of a menu_items row callers need.
type MenuItem struct {
	ID       uuid.UUID
	Name     string
	Category string
}

// MenuItemStore has no read/write methods yet — kept in place (rather than
// deleted alongside its former ListActive method) for the Financial Agent /
// Demand Forecasting work, which needs to read price/COGS/name from
// menu_items.
type MenuItemStore struct {
	pool *pgxpool.Pool
}

func NewMenuItemStore(pool *pgxpool.Pool) *MenuItemStore {
	return &MenuItemStore{pool: pool}
}
