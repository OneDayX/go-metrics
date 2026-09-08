// Package database provides a PostgreSQL connection pool used by the server.
package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

// New applies pending migrations and opens a connection pool. The pool is
// lazy, so use Ping to find out whether the database is reachable.
func New(ctx context.Context, dsn string) (*DB, error) {
	if err := migrateSchema(dsn); err != nil {
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return &DB{pool: pool}, nil
}

func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// Ping checks that the database accepts connections.
func (db *DB) Ping(ctx context.Context) error {
	if db == nil {
		return errors.New("database is not configured")
	}
	return db.pool.Ping(ctx)
}

func (db *DB) Close() {
	if db == nil {
		return
	}
	db.pool.Close()
}
