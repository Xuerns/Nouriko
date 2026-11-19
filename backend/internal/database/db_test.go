package database

import (
	"testing"

	"Nouriko/internal/config"

	"github.com/jackc/pgx/v5"
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
