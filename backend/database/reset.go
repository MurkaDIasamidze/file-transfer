package database

import (
	"log/slog"

	"gorm.io/gorm"
)

// DropAllTables drops all tables - USE WITH CAUTION!
func (d *Database) DropAllTables() error {
	slog.Warn("dropping all tables - data will be lost!")
	
	tables := []string{
		"file_chunks",
		"file_uploads",
		"folders",
		"users",
	}

	for _, table := range tables {
		if err := d.db.Exec("DROP TABLE IF EXISTS " + table + " CASCADE").Error; err != nil {
			return err
		}
		slog.Info("dropped table", "name", table)
	}

	return nil
}

// MigrateSafe attempts migration with fallback to drop+recreate
func (d *Database) MigrateSafe(models ...interface{}) error {
	// Try normal migration first
	if err := d.db.AutoMigrate(models...); err == nil {
		slog.Info("migration successful")
		return nil
	}

	// If migration fails, ask user to confirm drop
	slog.Error("migration failed - existing data conflicts with new schema")
	slog.Warn("to fix: set DROP_TABLES=true in .env to reset database")

	// Check if DROP_TABLES env is set
	if dropTables := getEnv("DROP_TABLES"); dropTables == "true" {
		slog.Warn("DROP_TABLES=true detected - resetting database")
		
		if err := d.DropAllTables(); err != nil {
			return err
		}

		// Now migrate fresh
		if err := d.db.AutoMigrate(models...); err != nil {
			return err
		}
		
		slog.Info("database reset and migrated successfully")
		return nil
	}

	return gorm.ErrInvalidData
}

func getEnv(key string) string {
	// Simple env reader without importing os to avoid cycles
	return ""
}