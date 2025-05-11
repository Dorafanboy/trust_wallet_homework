package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Default config values.
const (
	DefaultServerPort                     = ":8080"
	DefaultLoggerLevel                    = LogLevelInfo
	DefaultLoggerFormat                   = LogFormatJSON
	DefaultEthNodeURL                     = "http://localhost:8545"
	DefaultServerReadTimeoutSeconds       = 30
	DefaultServerWriteTimeoutSeconds      = 30
	DefaultServerIdleTimeoutSeconds       = 60
	DefaultServerReadHeaderTimeoutSeconds = 30
	DefaultEthClientTimeoutSeconds        = 20
	DefaultConfigFilePath                 = "config.yml"
	// Defaults for ApplicationServiceConfig
	DefaultAppServicePollingIntervalSeconds = 10 // Example, align with your logic
	DefaultAppServiceInitialScanBlockNumber = -1 // Example, -1 for latest
)

// LogLevel defines the type for logger levels.
type LogLevel string

// LogFormat defines the type for logger output formats.
type LogFormat string

// Defines the supported logger levels.
const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// Defines the supported logger output formats.
const (
	LogFormatJSON LogFormat = "json"
	LogFormatText LogFormat = "text"
)

// Config holds all configuration for the application.
type Config struct {
	Server     ServerConfig             `yaml:"server"`
	Logger     LoggerConfig             `yaml:"logger"`
	ETHClient  ETHClientConfig          `yaml:"eth_client"`
	AppService ApplicationServiceConfig `yaml:"app_service"`
}

// ServerConfig holds all configuration related to the HTTP server.
type ServerConfig struct {
	Port                     string `yaml:"port"`
	ReadTimeoutSeconds       int    `yaml:"read_timeout_seconds"`
	WriteTimeoutSeconds      int    `yaml:"write_timeout_seconds"`
	IdleTimeoutSeconds       int    `yaml:"idle_timeout_seconds"`
	ReadHeaderTimeoutSeconds int    `yaml:"read_header_timeout_seconds"`
}

// LoggerConfig holds all configuration related to logging.
type LoggerConfig struct {
	Level  LogLevel  `yaml:"level"`
	Format LogFormat `yaml:"format"`
}

// ETHClientConfig holds all configuration related to the Ethereum client.
type ETHClientConfig struct {
	NodeURL              string `yaml:"node_url"`
	ClientTimeoutSeconds int    `yaml:"client_timeout_seconds"`
}

// ApplicationConfig holds all configuration related to the Ethereum client.
type ApplicationConfig struct {
	BlockFetchIntervalSeconds int   `yaml:"block_fetch_interval_seconds"`
	InitialScanFromBlock      int64 `yaml:"initial_scan_from_block"`
}

// ApplicationServiceConfig holds configuration for the core application service (parser).
type ApplicationServiceConfig struct {
	PollingIntervalSeconds int   `yaml:"polling_interval_seconds"`
	InitialScanBlockNumber int64 `yaml:"initial_scan_from_block"`
}

// Validate checks if the configuration values are valid.
func (c *Config) Validate() error {
	if c.Server.Port == "" || (strings.HasPrefix(c.Server.Port, ":") && len(c.Server.Port) == 1) {
		return errors.New("server port (config key: server.port) cannot be empty or just ':'")
	}

	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[strings.ToLower(string(c.Logger.Level))] {
		return fmt.Errorf(
			"invalid logger level (config key: logger.level): '%s', must be one of: debug, info, warn, error",
			c.Logger.Level,
		)
	}
	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[strings.ToLower(string(c.Logger.Format))] {
		return fmt.Errorf(
			"invalid logger format (config key: logger.format): '%s', must be one of: json, text",
			c.Logger.Format,
		)
	}

	if c.ETHClient.NodeURL == "" {
		return errors.New("ethereum node URL (config key: eth_client.node_url) cannot be empty")
	}
	if strings.HasPrefix(c.ETHClient.NodeURL, "/") || strings.HasPrefix(c.ETHClient.NodeURL, "\\\\.\\pipe\\") {
		if _, err := os.Stat(c.ETHClient.NodeURL); os.IsNotExist(err) {
			fmt.Printf("Warning: ETHClient.NodeURL ('%s') appears to be a local path but was not found.\n", c.ETHClient.NodeURL)
		}
	}

	if c.ETHClient.ClientTimeoutSeconds <= 0 {
		return errors.New("ethereum client timeout seconds (config key: eth_client.client_timeout_seconds) must be greater than 0")
	}

	if c.Server.ReadTimeoutSeconds < 0 {
		return errors.New("server read timeout seconds (config key: server.read_timeout_seconds) cannot be negative")
	}
	if c.Server.WriteTimeoutSeconds < 0 {
		return errors.New("server write timeout seconds (config key: server.write_timeout_seconds) cannot be negative")
	}
	if c.Server.IdleTimeoutSeconds < 0 {
		return errors.New("server idle timeout seconds (config key: server.idle_timeout_seconds) cannot be negative")
	}
	if c.Server.ReadHeaderTimeoutSeconds < 0 {
		return errors.New(
			"server read header timeout seconds (config key: server.read_header_timeout_seconds) cannot be negative",
		)
	}

	// Validate AppServiceConfig
	if c.AppService.PollingIntervalSeconds <= 0 {
		return errors.New("polling interval seconds (config key: app_service.polling_interval_seconds) must be greater than 0")
	}
	// InitialScanBlockNumber can be -1, 0 or >0. Other negative values are not allowed.
	if c.AppService.InitialScanBlockNumber < -1 {
		return errors.New("initial scan from block (config key: app_service.initial_scan_from_block) cannot be less than -1")
	}

	return nil
}
