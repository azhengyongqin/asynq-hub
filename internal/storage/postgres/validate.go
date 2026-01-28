package postgres

import (
	"fmt"
	"net/url"
	"strings"
)

func validatePostgresURI(dsn string) error {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return fmt.Errorf("empty postgres dsn")
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("invalid postgres dsn: %w", err)
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return fmt.Errorf("postgres dsn must be URI with scheme postgres:// or postgresql:// (got %q)", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("postgres dsn missing host")
	}
	return nil
}
