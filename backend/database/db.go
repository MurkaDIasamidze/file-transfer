package database

import (
	"file-transfer-backend/config"
	"file-transfer-backend/models"
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type IDatabase interface {
	Connect() error
	GetDB() *gorm.DB
	Close() error
}

type Database struct {
	config *config.DatabaseConfig
	db     *gorm.DB
}

func NewDatabase(cfg *config.DatabaseConfig) IDatabase {
	return &Database{
		config: cfg,
	}
}

func (d *Database) Connect() error {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		d.config.Host,
		d.config.User,
		d.config.Password,
		d.config.Name,
		d.config.Port,
		d.config.SSLMode,
	)

	var err error
	d.db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connected successfully")

	// Auto migrate models
	if err = d.db.AutoMigrate(&models.FileUpload{}, &models.FileChunk{}); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database migration completed")
	return nil
}

func (d *Database) GetDB() *gorm.DB {
	return d.db
}

func (d *Database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}