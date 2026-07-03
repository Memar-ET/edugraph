// Command migrate applies Neo4j graph migrations (constraints, indexes,
// graph structure) from backend/db/neo4j/migrations. Postgres migrations
// are handled separately by flyway (see Makefile `make migrate`).
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/edugraph-ai/edugraph/pkg/config"
	"github.com/edugraph-ai/edugraph/pkg/database/neo4j"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "neo4j" {
		fmt.Fprintln(os.Stderr, "usage: migrate neo4j")
		os.Exit(1)
	}

	cfg := config.Load()
	driver, err := neo4j.NewDriver(cfg.Neo4j)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect neo4j: %v\n", err)
		os.Exit(1)
	}
	defer driver.Close(context.Background())

	if err := runMigrations(driver, "db/neo4j/migrations"); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("neo4j migrations applied")
}

func runMigrations(driver neo4jdriver.DriverWithContext, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".cypher") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	ctx := context.Background()
	session := driver.NewSession(ctx, neo4jdriver.SessionConfig{AccessMode: neo4jdriver.AccessModeWrite})
	defer session.Close(ctx)

	for _, name := range files {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		for _, stmt := range splitStatements(string(content)) {
			if _, err := session.Run(ctx, stmt, nil); err != nil {
				return fmt.Errorf("run %s: %w", name, err)
			}
		}
		fmt.Printf("applied %s\n", name)
	}
	return nil
}

// splitStatements turns a .cypher file into individual statements, dropping
// blank lines and `//` comments (Cypher migration files may contain several
// CREATE CONSTRAINT/INDEX statements separated by semicolons).
func splitStatements(content string) []string {
	var stmts []string
	for _, raw := range strings.Split(content, ";") {
		var lines []string
		for _, line := range strings.Split(raw, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "//") {
				continue
			}
			lines = append(lines, trimmed)
		}
		if stmt := strings.Join(lines, " "); stmt != "" {
			stmts = append(stmts, stmt)
		}
	}
	return stmts
}
