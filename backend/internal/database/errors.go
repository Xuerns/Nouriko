package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX abstracts pgxpool.Pool and pgx.Tx so repository methods can execute
// both standalone and inside transactions.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Sentinel errors returned by database repositories.
var (
	ErrNotFound              = errors.New("record not found")
	ErrConflict              = errors.New("record already exists")
	ErrTokenInvalidOrExpired = errors.New("token invalid, already used, or expired")
)

// pgUniqueViolationCode is PostgreSQL SQLSTATE 23505 (unique_violation).
const pgUniqueViolationCode = "23505"

// IsUniqueViolation reports whether err is a PostgreSQL unique_violation (SQLSTATE 23505).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolationCode
}
