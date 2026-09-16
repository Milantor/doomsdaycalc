package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// migrationsFS: SQL migrations compiled into the binary, so deploy stays a single
// file with no .sql files to copy next to it.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate applies every pending migration. Idempotent: goose tracks applied
// versions in its goose_db_version table.
// goose uses database/sql, so it gets a *sql.DB backed by the pgxpool through the
// stdlib adapter. Closing that wrapper does not close the pool.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	sqlDB := stdlib.OpenDBFromPool(pool)
	defer sqlDB.Close()

	// Embedded FS is rooted at this package directory and migrations sit one level
	// down, so re-root before handing it to goose.
	dir, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("sub migrations fs: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, dir)
	if err != nil {
		return fmt.Errorf("goose provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
