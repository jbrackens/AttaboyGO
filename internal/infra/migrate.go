package infra

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies all pending database migrations.
func RunMigrations(dsn string, logger *slog.Logger) error {
	migrationDir := findMigrationDir()
	sourceURL := fmt.Sprintf("file://%s", migrationDir)

	m, err := migrate.New(sourceURL, dsn)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}

	version, dirty, _ := m.Version()
	logger.Info("migrations applied", "version", version, "dirty", dirty)

	return nil
}

// findMigrationDir walks up from cwd looking for db/migrations.
func findMigrationDir() string {
	dir, _ := os.Getwd()
	for {
		candidate := dir + "/db/migrations"
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := dir[:max(0, len(dir)-1)]
		for parent != "" && parent[len(parent)-1] != '/' {
			parent = parent[:len(parent)-1]
		}
		if parent == "" || parent == "/" {
			break
		}
		dir = parent[:len(parent)-1]
	}
	return "db/migrations"
}
