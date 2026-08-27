package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// RunMigrations applies all .up.sql migrations in order.
// It uses a simple schema_migrations table to track applied migrations.
func RunMigrations(dbURL string) error {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	// Create tracking table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	// Read migration files from disk
	entries, err := os.ReadDir(filepath.Join(".", "migrations"))
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	// Collect .up.sql files
	var upFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			upFiles = append(upFiles, e.Name())
		}
	}
	sort.Strings(upFiles)

	// Get already applied
	applied := make(map[string]bool)
	rows, err := db.Query("SELECT filename FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("query migrations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var fn string
		if err := rows.Scan(&fn); err != nil {
			return err
		}
		applied[fn] = true
	}

	// Apply new migrations
	for _, fn := range upFiles {
		if applied[fn] {
			continue
		}
		content, err := os.ReadFile(filepath.Join("migrations", fn))
		if err != nil {
			return fmt.Errorf("read %s: %w", fn, err)
		}
		log.Printf("Applying migration: %s", fn)
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("execute %s: %w", fn, err)
		}
		if _, err := db.Exec("INSERT INTO schema_migrations (filename) VALUES ($1)", fn); err != nil {
			return fmt.Errorf("record %s: %w", fn, err)
		}
	}

	log.Println("Migrations complete")
	return nil
}

// RunSeedData applies seed .sql files in the migrations directory.
func RunSeedData(dbURL string) error {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf("open db for seed: %w", err)
	}
	defer db.Close()

	entries, err := os.ReadDir(filepath.Join(".", "migrations"))
	if err != nil {
		return fmt.Errorf("read migrations dir for seed: %w", err)
	}

	seedPrefixes := []string{"000011_seed_", "000012_seed_", "000013_seed_", "000014_seed_"}
	var seedFiles []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		for _, prefix := range seedPrefixes {
			if strings.HasPrefix(name, prefix) {
				seedFiles = append(seedFiles, name)
				break
			}
		}
	}
	sort.Strings(seedFiles)

	for _, fn := range seedFiles {
		content, err := os.ReadFile(filepath.Join("migrations", fn))
		if err != nil {
			return fmt.Errorf("read seed %s: %w", fn, err)
		}
		log.Printf("Applying seed: %s", fn)
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("execute seed %s: %w", fn, err)
		}
	}

	log.Println("Seed data complete")
	return nil
}
