package postgres

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ApplyMigrationsFromDir 以“按文件名排序”的方式执行 SQL 迁移。
// 说明：MVP 为了减少依赖，使用最朴素的 SQL 文件执行方式；后续可切换 goose/atlas。
func ApplyMigrationsFromDir(ctx context.Context, db *sql.DB, dir string) error {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var files []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		files = append(files, filepath.Join(dir, name))
	}
	sort.Strings(files)

	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, string(b)); err != nil {
			return err
		}
	}
	return nil
}
