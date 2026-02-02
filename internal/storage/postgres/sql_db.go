package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// OpenStdlib 仅用于 migrations（database/sql）。
// 使用 pgx 驱动，保持与 GORM 兼容
func OpenStdlib(dsn string) (*sql.DB, error) {
	if err := validatePostgresURI(dsn); err != nil {
		return nil, fmt.Errorf("invalid POSTGRES_DSN: %w", err)
	}
	return sql.Open("pgx", dsn)
}
