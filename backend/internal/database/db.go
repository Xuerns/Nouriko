// Package database manages PostgreSQL connections and embedded SQL migrations
// for both local Docker Compose PostgreSQL and production Supabase PostgreSQL.
package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"time"

	"Nouriko/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps a pgxpool.Pool with lifecycle helpers.
type DB struct {
	Pool *pgxpool.Pool
}

// BuildPoolConfig parses the DSN and applies mode-specific pgx settings.
// In production (Supabase), Supavisor transaction mode (port 6543) does not
// support named prepared statement caching across transactions, so we switch
// to pgx.QueryExecModeExec.
func BuildPoolConfig(cfg *config.Config) (*pgxpool.Config, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	poolCfg.MaxConns = cfg.Database.MaxConns
	poolCfg.MinConns = 1
	poolCfg.MaxConnLifetime = 30 * time.Minute
	poolCfg.MaxConnIdleTime = 5 * time.Minute
	poolCfg.HealthCheckPeriod = 30 * time.Second

	if cfg.IsProduction() || poolCfg.ConnConfig.Port == 6543 {
		poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	}

	return poolCfg, nil
}

// Connect opens a PostgreSQL connection pool and verifies connectivity.
func Connect(ctx context.Context, cfg *config.Config) (*DB, error) {
	poolCfg, err := BuildPoolConfig(cfg)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Ping checks whether the database is reachable.
func (d *DB) Ping(ctx context.Context) error {
	if d == nil || d.Pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}
	return d.Pool.Ping(ctx)
}

// Close releases all pooled database connections.
func (d *DB) Close() {
	if d != nil && d.Pool != nil {
		d.Pool.Close()
	}
}

// WithTx executes fn inside a PostgreSQL transaction on pool. If fn returns an
// error, the transaction is rolled back.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	if pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// RunMigrations applies embedded SQL migrations using goose.
func (d *DB) RunMigrations(ctx context.Context) error {
	if d == nil || d.Pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	sqlDB := stdlib.OpenDBFromPool(d.Pool)
	defer func() { _ = sqlDB.Close() }()

	subFS, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("load embedded migrations: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, subFS)
	if err != nil {
		return fmt.Errorf("init goose provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
