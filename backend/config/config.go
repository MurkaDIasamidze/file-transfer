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
}

type ServerConfig struct {
	Port           string
	MaxBodySize    int
	AllowedOrigins string
}

type DatabaseConfig struct {
	Host     string
	User     string
	Password string
	Name     string
	Port     string
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

func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:           getEnv("SERVER_PORT", "8080"),
			MaxBodySize:    getEnvInt("MAX_BODY_SIZE", 100*1024*1024),
			AllowedOrigins: getEnv("ALLOWED_ORIGINS", "*"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "driveclone"),
			Port:     getEnv("DB_PORT", "5432"),
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
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}