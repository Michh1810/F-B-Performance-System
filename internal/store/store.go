// Package store provides Postgres-backed persistence for menu items, TikTok
// trend snapshots, and the TikTok trend-signal corpus. It is the only place
// in the codebase that talks to the database; callers depend on it only
// through the narrow interfaces each consuming package declares (e.g.
// trend.SnapshotReader).
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
)

// Connect opens a connection pool to the given Postgres database URL,
// registers pgvector's types on every connection in the pool (required to
// read/write the `vector` column type), and verifies connectivity with a
// ping.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("store: parse database url: %w", err)
	}
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("store: create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping database: %w", err)
	}
	return pool, nil
}
