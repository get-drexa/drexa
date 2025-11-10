package database

import (
	"embed"

	"github.com/uptrace/bun/migrate"
)

//go:embed migrations/*.sql
var sqlMigrations embed.FS

// RunMigrations discovers and runs all migrations in the migrations directory.
// Currently, the migrations directory is in internal/db/migrations.
func RunMigrations() error {
	m := migrate.NewMigrations()
	return m.Discover(sqlMigrations)
}
