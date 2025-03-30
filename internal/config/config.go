package config

import (
	"os"

	"github.com/pelletier/go-toml/v2"
)

type AppConfig struct {
	PodsyncConfigPath   string `toml:"podsync_config_path"`
	DockerContainerName string `toml:"docker_container_name"`
	ServerPort          string `toml:"server_port"`
}

func LoadConfig(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg AppConfig
	err = toml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
