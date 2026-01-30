package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Upload   UploadConfig
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
	Directory       string
	ChunkSize       int
	MaxRetries      int
	VerifyInterval  int
}

func LoadConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:           getEnv("SERVER_PORT", "8080"),
			MaxBodySize:    getEnvAsInt("MAX_BODY_SIZE", 10*1024*1024), // 10MB default
			AllowedOrigins: getEnv("ALLOWED_ORIGINS", "*"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "filetransfer"),
			Port:     getEnv("DB_PORT", "5432"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Upload: UploadConfig{
			Directory:      getEnv("UPLOAD_DIR", "./uploads"),
			ChunkSize:      getEnvAsInt("CHUNK_SIZE", 1024*1024), // 1MB default
			MaxRetries:     getEnvAsInt("MAX_RETRIES", 3),
			VerifyInterval: getEnvAsInt("VERIFY_INTERVAL", 10),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}