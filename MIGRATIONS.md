# Database Migrations

This project uses a simple migration system to manage PostgreSQL database schema changes.

## Migration Files

Migration files are stored in the `migrations/` directory and follow this naming convention:
```
001_create_subscription_tables.sql
002_insert_sample_data.sql
003_add_new_feature.sql
```

## Current Migrations

1. **001_create_subscription_tables.sql** - Creates the initial subscription system tables
2. **002_insert_sample_data.sql** - Inserts sample data for testing

## Usage

### 1. Run All Pending Migrations
```bash
# Using the batch script
migrate.bat up

# Or using Go directly
go run cmd/migrate/main.go -command=up
```

### 2. Check Migration Status
```bash
# Using the batch script
migrate.bat status

# Or using Go directly  
go run cmd/migrate/main.go -command=status
```

### 3. Automatic Migrations
The main application automatically runs migrations on startup:
```bash
go run main.go
```

## Creating New Migrations

1. Create a new SQL file in the `migrations/` directory
2. Use the next sequential number (e.g., `003_add_user_profiles.sql`)
3. Write your SQL commands
4. Run `migrate.bat up` to apply

### Migration File Example:
```sql
-- Migration: Add User Profiles
-- Version: 003
-- Description: Add user profile information to subscribers

ALTER TABLE subscribers ADD COLUMN first_name VARCHAR(100);
ALTER TABLE subscribers ADD COLUMN last_name VARCHAR(100);
ALTER TABLE subscribers ADD COLUMN phone_number VARCHAR(20);

CREATE INDEX idx_subscribers_name ON subscribers(first_name, last_name);
```

## Migration Tracking

The system automatically tracks applied migrations in a `migrations` table:
- `version` - Migration version number
- `name` - Migration name/description  
- `applied_at` - When the migration was applied

## Configuration

Migrations use the same database configuration as the main application:
```env
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=your_password
DATABASE_NAME=ms_scheduling
DATABASE_SSL_MODE=disable
```

## Safety Features

- ✅ **Transactions** - Each migration runs in a transaction
- ✅ **Tracking** - Applied migrations are tracked to prevent re-running
- ✅ **Ordering** - Migrations run in sequential order
- ✅ **Rollback** - Failed migrations are automatically rolled back
- ✅ **Idempotent** - Safe to run multiple times

## Troubleshooting

### Migration Failed
If a migration fails:
1. Check the error message in the console
2. Fix the SQL syntax in the migration file
3. Run `migrate.bat up` again

### Reset Database (Development Only)
To start fresh in development:
1. Drop and recreate your database
2. Run `migrate.bat up` to apply all migrations

### Check What's Applied
Use `migrate.bat status` to see:
- Which migrations have been applied
- Which migrations are pending
- When each migration was applied

## Examples

### Running Migrations for the First Time:
```bash
PS> migrate.bat status
=== Migration Status ===
Applied migrations: 0
Pending migrations: 2

Pending:
  - 001 - create_subscription_tables
  - 002 - insert_sample_data

PS> migrate.bat up
Running migrations...
Migrations table created/verified
Applying 2 migrations...
✓ Applied migration: 001 - create_subscription_tables
✓ Applied migration: 002 - insert_sample_data
All migrations applied successfully
✓ Migrations completed successfully
```

### Checking Status After Migrations:
```bash
PS> migrate.bat status
=== Migration Status ===
Applied migrations: 2
Pending migrations: 0

Applied:
  ✓ 001 - create_subscription_tables (applied: 2025-10-10 15:30:45)
  ✓ 002 - insert_sample_data (applied: 2025-10-10 15:30:46)
```