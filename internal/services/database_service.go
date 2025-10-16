package services

import (
	"database/sql"
	"fmt"
	"log"
	"ms-scheduling/internal/migrations"
	"path/filepath"

	_ "github.com/lib/pq" // PostgreSQL driver
)

type DatabaseService struct {
	DB       *sql.DB
	migrator *migrations.Migrator
}

func NewDatabaseService(dsn string) (*DatabaseService, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	log.Printf("Successfully connected to database using DSN")

	// Initialize migrator
	migrationsDir := filepath.Join("migrations")
	migrator := migrations.NewMigrator(db, migrationsDir)

	return &DatabaseService{
		DB:       db,
		migrator: migrator,
	}, nil
}

func (d *DatabaseService) Close() error {
	return d.DB.Close()
}

// CheckConnection verifies database connectivity for health checks
func (d *DatabaseService) CheckConnection() error {
	return d.DB.Ping()
}

// RunMigrations applies all pending database migrations
func (d *DatabaseService) RunMigrations() error {
	return d.migrator.RunMigrations()
}

// MigrationStatus shows current migration status
func (d *DatabaseService) MigrationStatus() error {
	return d.migrator.Status()
}

// InitializeTables ensures the database tables are properly set up
// This is a compatibility method that runs migrations
func (d *DatabaseService) InitializeTables() error {
	log.Println("Initializing database tables via migrations")
	return d.RunMigrations()
}
