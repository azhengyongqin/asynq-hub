package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// OpenStdlib 仅用于 migrations（database/sql）。
func OpenStdlib(dsn string) (*sql.DB, error) {
	if err := validatePostgresURI(dsn); err != nil {
		return nil, fmt.Errorf("invalid POSTGRES_DSN: %w", err)
	}
	return sql.Open("pgx", dsn)
}
