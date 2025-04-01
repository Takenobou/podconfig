package config

import (
	"log"
	"os"
	"strconv"
)

// AppConfig holds environment-based configuration for the app.
type AppConfig struct {
	PodsyncConfigPath   string
	DockerContainerName string
	ServerPort          string
}

// LoadConfig loads configuration from environment variables, falling back to defaults.
// Minimal validation for SERVER_PORT and warns if the config file doesn’t exist.
func LoadConfig() *AppConfig {
	cfg := &AppConfig{
		PodsyncConfigPath:   os.Getenv("PODSYNC_CONFIG_PATH"),
		DockerContainerName: os.Getenv("DOCKER_CONTAINER_NAME"),
		ServerPort:          os.Getenv("SERVER_PORT"),
	}

	if cfg.PodsyncConfigPath == "" {
		cfg.PodsyncConfigPath = "../config.toml"
	}
	if cfg.DockerContainerName == "" {
		cfg.DockerContainerName = "podsync"
	}
	if cfg.ServerPort == "" {
		cfg.ServerPort = "8080"
	}

	portNum, err := strconv.Atoi(cfg.ServerPort)
	if err != nil || portNum < 1 || portNum > 65535 {
		log.Fatalf("Invalid SERVER_PORT: %s", cfg.ServerPort)
	}

	if _, err := os.Stat(cfg.PodsyncConfigPath); err != nil {
		log.Printf("WARNING: No podsync config file found at %s (error: %v)",
			cfg.PodsyncConfigPath, err)
	}

	return cfg
}
