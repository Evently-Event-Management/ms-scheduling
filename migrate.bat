@echo off
REM Migration Tool for MS-Scheduling
REM Usage: migrate.bat [up|status]

if "%1"=="up" (
    echo Running migrations...
    go run cmd/migrate/main.go -command=up
) else if "%1"=="status" (
    echo Checking migration status...
    go run cmd/migrate/main.go -command=status
) else (
    echo Usage:
    echo   migrate.bat up       - Run all pending migrations
    echo   migrate.bat status   - Show migration status
    echo.
    echo Examples:
    echo   migrate.bat up
    echo   migrate.bat status
)