package config

import (
	"os"
	"strconv"
)

type Config struct {
	Database DBConfig
	Server   ServerConfig
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
			Host: getEnv("SERVER_HOST", "localhost"),
			Port: getEnvAsIntValue("SERVER_PORT", 5432),
		},
	}
	return config, nil
}

func getEnv(key, defaultValue string) string {
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
