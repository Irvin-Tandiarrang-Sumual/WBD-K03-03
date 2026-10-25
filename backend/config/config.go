package config

import (
	"os"
	"strconv"
)

type Config struct {
	Database DBConfig
	Server   ServerConfig
	Security SecurityConfig
	Session SessionConfig
}

type ServerConfig struct {
	Host string
	Port int
}

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

type SecurityConfig struct {
	DatabaseCredentialKey string
}

type SessionConfig struct {
	CookieSecure bool
}

func LoadConfig() (*Config, error) {
	config := &Config{
		Database: DBConfig{
			Host:     getEnv("INTERNAL_DB_HOST", "localhost"),
			Port:     getEnvAsIntValue("INTERNAL_DB_PORT", 5432),
			User:     getEnv("INTERNAL_DB_USER", "postgresql"),
			Password: getEnv("INTERNAL_DB_PASSWORD", ""),
			Name:     getEnv("INTERNAL_DB_NAME", "tubes-wbd-3"),
		},

		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnvAsIntValue("SERVER_PORT", 8080),
		},

		Security: SecurityConfig{
			DatabaseCredentialKey: getEnv(
				"DATABASE_CREDENTIAL_KEY",
				"",
			),
		},

		Session: SessionConfig{
			CookieSecure: getEnvAsBoolValue("SESSION_COOKIE_SECURE", false),
		},
	}

	return config, nil
}

func getEnv(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}

func getEnvAsIntValue(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}

	return defaultValue
}

func getEnvAsBoolValue(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}

	return defaultValue
}