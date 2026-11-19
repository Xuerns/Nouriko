package database

import (
	"context"
	"errors"
	"testing"

	"Nouriko/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestBuildPoolConfigLocal(t *testing.T) {
	cfg := &config.Config{
		Mode: config.ModeLocal,
		Database: config.Database{
			URL:      "postgres://nouriko:secret@localhost:5432/nouriko?sslmode=disable",
			MaxConns: 8,
		},
	}

	poolCfg, err := BuildPoolConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if poolCfg.MaxConns != 8 {
		t.Fatalf("expected MaxConns=8, got %d", poolCfg.MaxConns)
	}
	if poolCfg.ConnConfig.DefaultQueryExecMode == pgx.QueryExecModeExec {
		t.Fatalf("expected default query exec mode for local port 5432, got exec")
	}
}

func TestBuildPoolConfigProductionSupavisor(t *testing.T) {
	cfg := &config.Config{
		Mode: config.ModeProduction,
		Database: config.Database{
			URL:      "postgres://postgres.proj:secret@aws-0-ap-southeast-1.pooler.supabase.com:6543/postgres?sslmode=require",
			MaxConns: 10,
		},
	}

	poolCfg, err := BuildPoolConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if poolCfg.ConnConfig.DefaultQueryExecMode != pgx.QueryExecModeExec {
		t.Fatalf("expected QueryExecModeExec for production Supabase, got %v", poolCfg.ConnConfig.DefaultQueryExecMode)
	}
}

func TestIsUniqueViolation(t *testing.T) {
	if !IsUniqueViolation(&pgconn.PgError{Code: "23505"}) {
		t.Fatal("expected SQLSTATE 23505 to be recognized as unique violation")
	}
	if IsUniqueViolation(&pgconn.PgError{Code: "23503"}) {
		t.Fatal("expected SQLSTATE 23503 not to be recognized as unique violation")
	}
	if IsUniqueViolation(errors.New("generic error")) {
		t.Fatal("expected generic error not to be recognized as unique violation")
	}
}

func TestWithTxNilPool(t *testing.T) {
	if err := WithTx(context.Background(), nil, func(_ pgx.Tx) error { return nil }); err == nil {
		t.Fatal("expected error when pool is nil")
	}
}
