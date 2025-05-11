// Package config implements application configuration loading and management.
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads configuration from a YAML file, falling back to defaults.
func LoadConfig(filePath string) (*Config, error) {
	cfg := Config{
		Server: ServerConfig{
			Port:                     DefaultServerPort,
			ReadTimeoutSeconds:       DefaultServerReadTimeoutSeconds,
			WriteTimeoutSeconds:      DefaultServerWriteTimeoutSeconds,
			IdleTimeoutSeconds:       DefaultServerIdleTimeoutSeconds,
			ReadHeaderTimeoutSeconds: DefaultServerReadHeaderTimeoutSeconds,
		},
		Logger: LoggerConfig{
			Level:  DefaultLoggerLevel,
			Format: DefaultLoggerFormat,
		},
		ETHClient: ETHClientConfig{
			NodeURL:              DefaultEthNodeURL,
			ClientTimeoutSeconds: DefaultEthClientTimeoutSeconds,
		},
		AppService: ApplicationServiceConfig{
			PollingIntervalSeconds: DefaultAppServicePollingIntervalSeconds,
			InitialScanBlockNumber: DefaultAppServiceInitialScanBlockNumber,
		},
	}

	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Info: Config file '%s' not found, using default values for all settings.\n", filePath)
			if validationErr := cfg.Validate(); validationErr != nil {
				return nil, fmt.Errorf("default configuration validation failed: %w", validationErr)
			}
			return &cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file '%s': %w", filePath, err)
	}

	if err := yaml.Unmarshal(fileBytes, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config file '%s': %w", filePath, err)
	}

	if cfg.Server.Port != "" && !strings.HasPrefix(cfg.Server.Port, ":") {
		cfg.Server.Port = ":" + cfg.Server.Port
	} else if cfg.Server.Port == "" {
		cfg.Server.Port = DefaultServerPort
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("loaded configuration validation failed: %w", err)
	}

	fmt.Printf("Info: Configuration successfully loaded from '%s'.\n", filePath)
	return &cfg, nil
}
