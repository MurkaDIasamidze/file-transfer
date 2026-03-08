// config/config.go
package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Upload   UploadConfig
	JWT      JWTConfig
	S3       S3Config       // ← добавили
}

type ServerConfig struct {
	Port           string
	MaxBodySize    int
	AllowedOrigins string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type UploadConfig struct {
	Directory      string
	ChunkSize      int
	MaxRetries     int
	VerifyInterval int
}

type JWTConfig struct {
	Secret      string
	ExpiryHours int
}

// S3Config содержит все настройки для AWS S3
type S3Config struct {
	Region          string // eu-central-1
	AccessKeyID     string // AKIA...
	SecretAccessKey string // wJal...
	Bucket          string // driveclone-files
	// Если true — файлы хранятся в S3.
	// Если false — хранятся локально (для локальной разработки без AWS).
	Enabled bool
}

func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:           getEnv("SERVER_PORT", "8080"),
			MaxBodySize:    getEnvInt("MAX_BODY_SIZE", 100*1024*1024),
			AllowedOrigins: getEnv("ALLOWED_ORIGINS", "*"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "driveclone"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Upload: UploadConfig{
			Directory:      getEnv("UPLOAD_DIR", "./uploads"),
			ChunkSize:      getEnvInt("CHUNK_SIZE", 1024*1024),
			MaxRetries:     getEnvInt("MAX_RETRIES", 3),
			VerifyInterval: getEnvInt("VERIFY_INTERVAL", 10),
		},
		JWT: JWTConfig{
			Secret:      getEnv("JWT_SECRET", "change-me-in-production"),
			ExpiryHours: getEnvInt("JWT_EXPIRY_HOURS", 72),
		},
		S3: S3Config{
			Region:          getEnv("AWS_REGION", "eu-central-1"),
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
			Bucket:          getEnv("AWS_S3_BUCKET", ""),
			// S3 включается автоматически если все ключи заполнены
			Enabled: getEnv("AWS_ACCESS_KEY_ID", "") != "" &&
				getEnv("AWS_S3_BUCKET", "") != "",
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}