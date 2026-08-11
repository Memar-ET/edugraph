package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/edugraph-ai/edugraph/pkg/config"
	"github.com/edugraph-ai/edugraph/pkg/database/neo4j"
)

// blockCommentRegex matches /* ... */ across multiple lines.
// (?s) enables "dotall" mode so the dot (.) matches newlines.
var blockCommentRegex = regexp.MustCompile(`(?s)/\*.*?\*/`)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "neo4j" {
		fmt.Fprintln(os.Stderr, "usage: migrate neo4j")
		os.Exit(1)
	}

	cfg := config.Load()
	fmt.Println("NEO4J URI =", cfg.Neo4j.URI)
	fmt.Println("NEO4J USER =", cfg.Neo4j.User)
	// fmt.Println("NEO4J PASSWORD =", cfg.Neo4j.Password) // Hidden for security

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
	fmt.Println("✅ neo4j migrations applied successfully")
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

		statements := splitStatements(string(content))
		for i, stmt := range statements {
			if _, err := session.Run(ctx, stmt, nil); err != nil {
				return fmt.Errorf("run %s (statement %d): %w\nQuery: %s", name, i+1, err, stmt)
			}
		}
		fmt.Printf("✅ applied %s (%d statements)\n", name, len(statements))
	}
	return nil
}

// splitStatements turns a .cypher file into individual executable statements.
// It safely removes block comments /* ... */, single-line comments //, and inline comments.
func splitStatements(content string) []string {
	// 1. Strip block comments /* ... */
	content = blockCommentRegex.ReplaceAllString(content, "")

	var stmts []string
	// 2. Split by semicolon
	for _, raw := range strings.Split(content, ";") {
		var lines []string
		for _, line := range strings.Split(raw, "\n") {
			trimmed := strings.TrimSpace(line)

			// 3. Skip empty lines and full-line comments
			if trimmed == "" || strings.HasPrefix(trimmed, "//") {
				continue
			}

			// 4. Strip inline // comments so they don't break multi-line joins
			if idx := strings.Index(trimmed, "//"); idx != -1 {
				trimmed = strings.TrimSpace(trimmed[:idx])
			}

			if trimmed != "" {
				lines = append(lines, trimmed)
			}
		}

		stmt := strings.Join(lines, " ")
		stmt = strings.TrimSpace(stmt)

		// 5. Only add if there's actual Cypher code left
		if stmt != "" {
			stmts = append(stmts, stmt)
		}
	}
	return stmts
}
