//go:build integration

package testutil

import (
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// newMigrate wraps migrate.New for integration tests.
func newMigrate(sourceURL, databaseURL string) (*migrate.Migrate, error) {
	return migrate.New(sourceURL, databaseURL)
}
