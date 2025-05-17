package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migration represents a database migration
type Migration struct {
	Version     string
	Description string
	SQL         string
}

// RunMigrations executes all pending migrations
func (db *DB) RunMigrations(ctx context.Context) error {
	// Create migrations table if it doesn't exist
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	appliedMigrations, err := db.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Get available migrations
	availableMigrations, err := db.getAvailableMigrations()
	if err != nil {
		return fmt.Errorf("failed to get available migrations: %w", err)
	}

	// Apply pending migrations
	for _, migration := range availableMigrations {
		// Skip if already applied
		if contains(appliedMigrations, migration.Version) {
			continue
		}

		// Begin transaction
		tx, err := db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		// Execute migration
		_, err = tx.Exec(ctx, migration.SQL)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to execute migration %s: %w", migration.Version, err)
		}

		// Update migrations table (only if not already handled in the migration SQL)
		if !strings.Contains(migration.SQL, "INSERT INTO schema_migrations") {
			_, err = tx.Exec(ctx, `
				INSERT INTO schema_migrations (version) VALUES ($1)
			`, migration.Version)
			if err != nil {
				tx.Rollback(ctx)
				return fmt.Errorf("failed to record migration %s: %w", migration.Version, err)
			}
		}

		// Commit transaction
		err = tx.Commit(ctx)
		if err != nil {
			return fmt.Errorf("failed to commit transaction for migration %s: %w", migration.Version, err)
		}

		fmt.Printf("Applied migration: %s - %s\n", migration.Version, migration.Description)
	}

	return nil
}

// getAppliedMigrations returns a list of already applied migrations
func (db *DB) getAppliedMigrations(ctx context.Context) ([]string, error) {
	query := `SELECT version FROM schema_migrations ORDER BY version`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []string
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		migrations = append(migrations, version)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return migrations, nil
}

// getAvailableMigrations returns a list of available migrations from the embedded filesystem
func (db *DB) getAvailableMigrations() ([]Migration, error) {
	var migrations []Migration

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content, err := fs.ReadFile(migrationsFS, path.Join("migrations", entry.Name()))
		if err != nil {
			return nil, err
		}

		// Extract version and description from filename
		version := strings.TrimSuffix(entry.Name(), ".sql")
		description := strings.ReplaceAll(version, "_", " ")

		migrations = append(migrations, Migration{
			Version:     version,
			Description: description,
			SQL:         string(content),
		})
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// Helper function to check if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
