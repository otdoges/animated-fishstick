package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config contains all application configuration
type Config struct {
	Database DatabaseConfig
	Auth     AuthConfig
	CORS     CORSConfig
	Server   ServerConfig
}

// DatabaseConfig holds database connection parameters
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	Pool     struct {
		MaxConn     int
		MaxIdle     int
		MaxLifetime int
	}
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	ClerkPublishableKey string
	ClerkSecretKey      string
	JWTPublicKey        string
}

// CORSConfig holds CORS settings
type CORSConfig struct {
	AllowOrigins string
}

// ServerConfig holds server specific configuration
type ServerConfig struct {
	Port string
}

// Load loads configuration from environment variables or .env file
func Load() (*Config, error) {
	// Load .env file if it exists
	godotenv.Load()

	// Initialize with default values
	config := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "supabase"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Auth: AuthConfig{
			ClerkPublishableKey: getEnv("CLERK_PUBLISHABLE_KEY", ""),
			ClerkSecretKey:      getEnv("CLERK_SECRET_KEY", ""),
			JWTPublicKey:        getEnv("JWT_PUBLIC_KEY", ""),
		},
		CORS: CORSConfig{
			AllowOrigins: getEnv("CORS_ALLOW_ORIGINS", "*"),
		},
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
		},
	}

	// Parse database pool settings
	config.Database.Pool.MaxConn = getEnvAsInt("DB_MAX_CONNECTIONS", 10)
	config.Database.Pool.MaxIdle = getEnvAsInt("DB_MAX_IDLE_CONNECTIONS", 5)
	config.Database.Pool.MaxLifetime = getEnvAsInt("DB_MAX_CONNECTION_LIFETIME", 30)

	return config, nil
}

// Helper function to get an environment variable or a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// Helper function to get an environment variable as an integer
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	value, err := parseInt(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// Helper function to parse an integer
func parseInt(valueStr string) (int, error) {
	var value int
	_, err := fmt.Sscanf(valueStr, "%d", &value)
	return value, err
}
