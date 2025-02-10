package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	ManifestDir  string   `mapstructure:"manifest_dir"`
	BlobDir      string   `mapstructure:"blob_dir"`
	Destinations []string `mapstructure:"destinations"`
}

func LoadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("unable to determine user home directory: %w", err)
	}

	// Use a standard config directory.
	configDir := filepath.Join(homeDir, ".config", "lmstudio-ollama-bridge")
	configFile := filepath.Join(configDir, "config.yaml")

	// If the config file does not exist, create it with default values.
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}

		// Define default configuration.
		defaultConfig := []byte(fmt.Sprintf(`manifest_dir: "%s"
blob_dir: "%s"
destinations:
  - "%s"
`, filepath.Join(homeDir, ".ollama", "models", "manifests", "registry.ollama.ai"),
			filepath.Join(homeDir, ".ollama", "models", "blobs"),
			filepath.Join(homeDir, ".cache", "lm-studio", "ollama"),
		))

		if err := os.WriteFile(configFile, defaultConfig, 0644); err != nil {
			return nil, fmt.Errorf("failed to write default config file: %w", err)
		}
	}

	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	return &cfg, nil
}
