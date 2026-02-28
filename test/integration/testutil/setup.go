//go:build integration

package testutil

import (
	"context"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/attaboy/platform/internal/app"
	"github.com/attaboy/platform/internal/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	TestJWTSecret          = "integration-test-secret"
	TestStripeWebhookSecret = "whsec_test_integration_secret"
	TestDBHost             = "localhost"
	TestDBPort             = 5435
	TestDBUser             = "attaboy"
	TestDBPass             = "attaboy"
	TestDBName             = "attaboy_test"
)

// TestEnv holds all resources for an integration test.
type TestEnv struct {
	Server *httptest.Server
	Pool   *pgxpool.Pool
	JWTMgr *auth.JWTManager
	t      *testing.T
}

var (
	sharedPool *pgxpool.Pool
	poolOnce   sync.Once
	poolErr    error
)

func testDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		TestDBUser, TestDBPass, TestDBHost, TestDBPort, TestDBName)
}

func bootstrapDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		TestDBUser, TestDBPass, TestDBHost, TestDBPort, "attaboy")
}

func ensureTestDB() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to the main database to create the test database
	bPool, err := pgxpool.New(ctx, bootstrapDSN())
	if err != nil {
		return fmt.Errorf("connect bootstrap db: %w", err)
	}
	defer bPool.Close()

	var exists bool
	err = bPool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", TestDBName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check db exists: %w", err)
	}

	if !exists {
		_, err = bPool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", TestDBName))
		if err != nil {
			return fmt.Errorf("create test db: %w", err)
		}
	}

	return nil
}

func runMigrations(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check if the citext extension exists
	_, err := pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS citext")
	if err != nil {
		return fmt.Errorf("create citext extension: %w", err)
	}

	// Read and apply migrations from db/migrations
	// We use the golang-migrate library approach but since tests need a pool,
	// we use migrate directly
	dsn := testDSN()

	// Find the project root by looking for go.mod
	projectRoot := findProjectRoot()

	migratePath := fmt.Sprintf("file://%s/db/migrations", projectRoot)

	m, err := newMigrate(migratePath, dsn)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err.Error() != "no change" {
		return fmt.Errorf("migrate up: %w", err)
	}

	return nil
}

func findProjectRoot() string {
	// Walk up from current working directory looking for go.mod
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(dir + "/go.mod"); err == nil {
			return dir
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
	// Fallback: assume we're inside the project
	return "."
}

func getSharedPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	poolOnce.Do(func() {
		if err := ensureTestDB(); err != nil {
			poolErr = err
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		poolCfg, err := pgxpool.ParseConfig(testDSN())
		if err != nil {
			poolErr = fmt.Errorf("parse pool config: %w", err)
			return
		}
		poolCfg.MaxConns = 10
		poolCfg.MinConns = 1

		sharedPool, err = pgxpool.NewWithConfig(ctx, poolCfg)
		if err != nil {
			poolErr = fmt.Errorf("create pool: %w", err)
			return
		}

		if err := runMigrations(sharedPool); err != nil {
			poolErr = fmt.Errorf("run migrations: %w", err)
			sharedPool.Close()
			sharedPool = nil
			return
		}
	})

	if poolErr != nil {
		t.Fatalf("failed to initialize test pool: %v", poolErr)
	}
	return sharedPool
}

// NewTestEnv creates a test environment with an httptest.Server backed by the real router and test DB.
func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	pool := getSharedPool(t)

	jwtMgr := auth.NewJWTManager(TestJWTSecret, 24*time.Hour, 8*time.Hour, 12*time.Hour)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	router := app.NewRouter(app.RouterDeps{
		Pool:                pool,
		JWTMgr:              jwtMgr,
		Logger:              logger,
		StripeSecretKey:     "",
		StripeWebhookSecret: TestStripeWebhookSecret,
		RandomOrgAPIKey:     "",
		SlotopolBaseURL:     "http://localhost:4002",
		CORSAllowedOrigins:  "*",
	})

	server := httptest.NewServer(router)

	env := &TestEnv{
		Server: server,
		Pool:   pool,
		JWTMgr: jwtMgr,
		t:      t,
	}

	t.Cleanup(func() {
		server.Close()
		env.CleanAll()
	})

	// Clean before test to ensure isolation
	env.CleanAll()

	return env
}
