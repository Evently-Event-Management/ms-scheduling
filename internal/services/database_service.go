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

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func NewDatabaseService(config DatabaseConfig) (*DatabaseService, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	log.Printf("Successfully connected to database: %s", config.DBName)

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

// RunMigrations applies all pending database migrations
func (d *DatabaseService) RunMigrations() error {
	return d.migrator.RunMigrations()
}

// MigrationStatus shows current migration status
func (d *DatabaseService) MigrationStatus() error {
	return d.migrator.Status()
}

// InitializeTables is now deprecated in favor of migrations
func (d *DatabaseService) InitializeTables() error {
	// Use migrations instead
	return d.RunMigrations()
}
