package services

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// beginTx is a helper to start a transaction with defer rollback.
// Usage:
//
//	tx, err := beginTx(ctx, pool)
//	if err != nil { return err }
//	defer tx.Rollback(ctx)
//	// ... do work ...
//	return tx.Commit(ctx)
func beginTx(ctx context.Context, pool *pgxpool.Pool) (pgx.Tx, error) {
	return pool.Begin(ctx)
}
