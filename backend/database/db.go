package database

import (
	"file-transfer-backend/config"
	"file-transfer-backend/models"
	"fmt"
	"log/slog"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type IDatabase interface {
	Connect() error
	GetDB() *gorm.DB
	Close() error
}

type Database struct {
	cfg *config.DatabaseConfig
	db  *gorm.DB
}

func New(cfg *config.DatabaseConfig) IDatabase {
	return &Database{cfg: cfg}
}

func (d *Database) Connect() error {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		d.cfg.Host, d.cfg.User, d.cfg.Password,
		d.cfg.Name, d.cfg.Port, d.cfg.SSLMode,
	)

	var err error
	d.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	slog.Info("database connected", "host", d.cfg.Host, "name", d.cfg.Name)

	if err = d.db.AutoMigrate(
		&models.User{},
		&models.Folder{},
		&models.FileUpload{},
		&models.FileChunk{},
	); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}

	slog.Info("database migration completed")
	return nil
}

func (d *Database) GetDB() *gorm.DB { return d.db }

func (d *Database) Close() error {
	sql, err := d.db.DB()
	if err != nil {
		return err
	}
	return sql.Close()
}