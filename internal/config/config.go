package config

import "os"

type AppConfig struct {
	PodsyncConfigPath   string
	DockerContainerName string
	ServerPort          string
}

// LoadConfig loads configuration from environment variables, falling back to defaults.
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

	return cfg
}
