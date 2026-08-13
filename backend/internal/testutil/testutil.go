// Package testutil provides shared setup for integration tests that hit
// a real Postgres/Neo4j -- checklist 12.1's tests intentionally exercise
// real database operations rather than mocking the pool, matching how
// every other part of this session's work was verified. Connection
// details default to match .github/workflows/ci.yml's backend-test job
// service containers, overridable via the same POSTGRES_*/NEO4J_* env
// vars pkg/config already uses, so the same tests run unchanged locally
// against a docker-run postgres/neo4j.
package testutil

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// RequirePostgres returns a pool to a real Postgres with every migration
// applied, or skips the test (not fails) if no database is reachable --
// so `go test ./...` still passes for a contributor without Docker
// running, while CI (which always has the service container up) always
// exercises these for real.
func RequirePostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()

	host := envOr("POSTGRES_HOST", "localhost")
	port := envOr("POSTGRES_PORT", "5432")
	db := envOr("POSTGRES_DB", "edugraph_test")
	user := envOr("POSTGRES_USER", "edugraph")
	password := envOr("POSTGRES_PASSWORD", "testpass")
	sslmode := envOr("POSTGRES_SSLMODE", "disable")

	dsn := "postgres://" + user + ":" + password + "@" + host + ":" + port + "/" + db + "?sslmode=" + sslmode

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("postgres not available (%v) -- start it to run this test (see .github/workflows/ci.yml's backend-test service containers for the expected setup)", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("postgres not reachable at %s:%s (%v) -- start it to run this test", host, port, err)
	}

	applyMigrations(t, pool)
	return pool
}

// RequireNeo4j mirrors RequirePostgres for tests that also need the
// graph mirror (e.g. curriculum ApproveAndPromote).
func RequireNeo4j(t *testing.T) neo4jdriver.DriverWithContext {
	t.Helper()

	uri := envOr("NEO4J_URI", "bolt://localhost:7687")
	user := envOr("NEO4J_USER", "neo4j")
	password := envOr("NEO4J_PASSWORD", "testpass")

	driver, err := neo4jdriver.NewDriverWithContext(uri, neo4jdriver.BasicAuth(user, password, ""))
	if err != nil {
		t.Skipf("neo4j driver init failed (%v) -- start it to run this test", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		_ = driver.Close(ctx)
		t.Skipf("neo4j not reachable at %s (%v) -- start it to run this test", uri, err)
	}

	t.Cleanup(func() { _ = driver.Close(context.Background()) })
	return driver
}

// migrationsOnce ensures the (possibly slow, many-statement) migration
// apply only happens once per test binary run, not once per test
// function -- every test in the package shares the same already-migrated
// database.
var migrationsOnce sync.Once

func applyMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	migrationsOnce.Do(func() {
		ctx := context.Background()
		_, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS _test_migrations_applied (filename TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`)
		if err != nil {
			t.Fatalf("create migration tracker: %v", err)
		}

		dir := migrationsDir(t)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read migrations dir %s: %v", dir, err)
		}
		var files []string
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
				files = append(files, e.Name())
			}
		}
		sort.Strings(files) // V001, V002, ... sorts correctly as plain strings

		for _, f := range files {
			var already bool
			err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM _test_migrations_applied WHERE filename = $1)`, f).Scan(&already)
			if err != nil {
				t.Fatalf("check migration %s: %v", f, err)
			}
			if already {
				continue
			}

			sqlBytes, err := os.ReadFile(filepath.Join(dir, f))
			if err != nil {
				t.Fatalf("read migration %s: %v", f, err)
			}
			if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
				t.Fatalf("apply migration %s: %v", f, err)
			}
			if _, err := pool.Exec(ctx, `INSERT INTO _test_migrations_applied (filename) VALUES ($1)`, f); err != nil {
				t.Fatalf("record migration %s: %v", f, err)
			}
		}
	})
}

// migrationsDir resolves backend/db/migrations relative to this source
// file (runtime.Caller), not the test binary's cwd -- go test runs with
// cwd set to the package under test, which varies per caller, but this
// file's own location on disk never does.
func migrationsDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve testutil.go's own path")
	}
	// this file: backend/internal/testutil/testutil.go -> backend/db/migrations
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "db", "migrations")
}
