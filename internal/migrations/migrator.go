package migrations

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Migrator struct {
	DB            *sql.DB
	MigrationsDir string
}

type Migration struct {
	Version   string
	Name      string
	FilePath  string
	AppliedAt *time.Time
}

func NewMigrator(db *sql.DB, migrationsDir string) *Migrator {
	return &Migrator{
		DB:            db,
		MigrationsDir: migrationsDir,
	}
}

// CreateMigrationsTable creates the migrations tracking table
func (m *Migrator) CreateMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS migrations (
			version VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP DEFAULT NOW()
		)
	`
	_, err := m.DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}
	log.Println("Migrations table created/verified")
	return nil
}

// GetAppliedMigrations returns a list of applied migrations
func (m *Migrator) GetAppliedMigrations() (map[string]Migration, error) {
	query := `SELECT version, name, applied_at FROM migrations ORDER BY version`
	rows, err := m.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %v", err)
	}
	defer rows.Close()

	applied := make(map[string]Migration)
	for rows.Next() {
		var migration Migration
		err := rows.Scan(&migration.Version, &migration.Name, &migration.AppliedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration: %v", err)
		}
		applied[migration.Version] = migration
	}
	return applied, nil
}

// GetPendingMigrations returns migrations that need to be applied
func (m *Migrator) GetPendingMigrations() ([]Migration, error) {
	// Get applied migrations
	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return nil, err
	}

	// Read migration files
	files, err := filepath.Glob(filepath.Join(m.MigrationsDir, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("failed to read migration files: %v", err)
	}

	var pending []Migration
	for _, file := range files {
		filename := filepath.Base(file)
		version := extractVersionFromFilename(filename)
		name := extractNameFromFilename(filename)

		if _, exists := applied[version]; !exists {
			pending = append(pending, Migration{
				Version:  version,
				Name:     name,
				FilePath: file,
			})
		}
	}

	// Sort by version
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Version < pending[j].Version
	})

	return pending, nil
}

// RunMigrations applies all pending migrations
func (m *Migrator) RunMigrations() error {
	// Create migrations table
	if err := m.CreateMigrationsTable(); err != nil {
		return err
	}

	// Get pending migrations
	pending, err := m.GetPendingMigrations()
	if err != nil {
		return err
	}

	if len(pending) == 0 {
		log.Println("No pending migrations to apply")
		return nil
	}

	log.Printf("Applying %d migrations...", len(pending))

	// Apply each migration
	for _, migration := range pending {
		if err := m.applyMigration(migration); err != nil {
			return fmt.Errorf("failed to apply migration %s: %v", migration.Version, err)
		}
		log.Printf("✓ Applied migration: %s - %s", migration.Version, migration.Name)
	}

	log.Println("All migrations applied successfully")
	return nil
}

// applyMigration applies a single migration
func (m *Migrator) applyMigration(migration Migration) error {
	// Read migration file
	content, err := ioutil.ReadFile(migration.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %v", err)
	}

	// Start transaction
	tx, err := m.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	_, err = tx.Exec(string(content))
	if err != nil {
		return fmt.Errorf("failed to execute migration SQL: %v", err)
	}

	// Record migration as applied
	_, err = tx.Exec(
		`INSERT INTO migrations (version, name) VALUES ($1, $2)`,
		migration.Version, migration.Name,
	)
	if err != nil {
		return fmt.Errorf("failed to record migration: %v", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %v", err)
	}

	return nil
}

// extractVersionFromFilename extracts version from filename like "001_initial_schema.sql"
func extractVersionFromFilename(filename string) string {
	parts := strings.Split(filename, "_")
	if len(parts) > 0 {
		return parts[0]
	}
	return filename
}

// extractNameFromFilename extracts name from filename like "001_initial_schema.sql"
func extractNameFromFilename(filename string) string {
	name := strings.TrimSuffix(filename, ".sql")
	parts := strings.Split(name, "_")
	if len(parts) > 1 {
		return strings.Join(parts[1:], "_")
	}
	return name
}

// Status shows migration status
func (m *Migrator) Status() error {
	if err := m.CreateMigrationsTable(); err != nil {
		return err
	}

	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return err
	}

	pending, err := m.GetPendingMigrations()
	if err != nil {
		return err
	}

	fmt.Println("\n=== Migration Status ===")
	fmt.Printf("Applied migrations: %d\n", len(applied))
	fmt.Printf("Pending migrations: %d\n", len(pending))

	if len(applied) > 0 {
		fmt.Println("\nApplied:")
		for _, migration := range applied {
			fmt.Printf("  ✓ %s - %s (applied: %s)\n",
				migration.Version, migration.Name, migration.AppliedAt.Format("2006-01-02 15:04:05"))
		}
	}

	if len(pending) > 0 {
		fmt.Println("\nPending:")
		for _, migration := range pending {
			fmt.Printf("  - %s - %s\n", migration.Version, migration.Name)
		}
	}

	return nil
}
