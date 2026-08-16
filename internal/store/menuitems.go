package store

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"fbperformance/internal/agents/financial"
	"fbperformance/internal/agents/menuidea"
)

// MenuItemStore reads menu_items rows. It implements
// financial.MenuItemLookup.
type MenuItemStore struct {
	pool *pgxpool.Pool
}

func NewMenuItemStore(pool *pgxpool.Pool) *MenuItemStore {
	return &MenuItemStore{pool: pool}
}

// Get loads a menu item's name and pricing, converting the DECIMAL dollar
// columns (current_price, cogs) into integer cents. It implements
// financial.MenuItemLookup.
func (s *MenuItemStore) Get(ctx context.Context, id uuid.UUID) (financial.MenuItem, error) {
	var item financial.MenuItem
	var priceDollars, cogsDollars float64
	err := s.pool.QueryRow(ctx, `SELECT name, current_price, cogs FROM menu_items WHERE id = $1`, id).
		Scan(&item.Name, &priceDollars, &cogsDollars)
	if err != nil {
		return financial.MenuItem{}, err
	}
	item.PriceCents = int64(math.Round(priceDollars * 100))
	item.EstimatedCOGSCents = int64(math.Round(cogsDollars * 100))
	return item, nil
}

// ListActive returns all active menu items' id, name, and category — the
// minimal shape the Menu Idea Agent needs to build a query embedding per
// item.
func (s *MenuItemStore) ListActive(ctx context.Context) ([]menuidea.MenuItem, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, category FROM menu_items WHERE is_active = true ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("store: list active menu items: %w", err)
	}
	defer rows.Close()

	var items []menuidea.MenuItem
	for rows.Next() {
		var item menuidea.MenuItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Category); err != nil {
			return nil, fmt.Errorf("store: scan menu item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list active menu items: %w", err)
	}
	return items, nil
}
