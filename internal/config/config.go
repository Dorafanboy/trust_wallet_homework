// Package config implements application configuration loading and management.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Default values.
const (
	DefaultConfigFile        = "config/config.yml"
	DefaultServerPort        = ":8080"
	DefaultEthereumRPCURL    = "https://cloudflare-eth.com"
	DefaultParserPollingInt  = 15
	DefaultParserInitialScan = -1
)

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	Port string `yaml:"port"`
}

// EthereumConfig holds Ethereum client-related configuration.
type EthereumConfig struct {
	RPCURL string `yaml:"rpc_url"`
}

// ParserConfig holds parser-specific configuration.
type ParserConfig struct {
	PollingIntervalSeconds int   `yaml:"polling_interval_seconds"`
	InitialScanBlockNumber int64 `yaml:"initial_scan_block_number"`
}

// Config holds the application configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Ethereum EthereumConfig `yaml:"ethereum"`
	Parser   ParserConfig   `yaml:"parser"`
}

// LoadConfig loads the configuration from a YAML file.
func LoadConfig(filePath string) (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port: DefaultServerPort,
		},
		Ethereum: EthereumConfig{
			RPCURL: DefaultEthereumRPCURL,
		},
		Parser: ParserConfig{
			PollingIntervalSeconds: DefaultParserPollingInt,
			InitialScanBlockNumber: DefaultParserInitialScan,
		},
	}

	loadPath := filePath
	if loadPath == "" {
		loadPath = DefaultConfigFile
	}

	fileBytes, err := os.ReadFile(loadPath)
	if err != nil {
		if os.IsNotExist(err) && (filePath == "" || filePath == DefaultConfigFile) {
			fmt.Printf("Config file '%s' not found, using default values for all sections.\n", loadPath)
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file '%s': %w", loadPath, err)
	}

	type partialConfig struct {
		Server   *ServerConfig   `yaml:"server"`
		Ethereum *EthereumConfig `yaml:"ethereum"`
		Parser   *ParserConfig   `yaml:"parser"`
	}
	var pCfg partialConfig

	if err := yaml.Unmarshal(fileBytes, &pCfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file '%s': %w", loadPath, err)
	}

	if pCfg.Server != nil {
		if pCfg.Server.Port != "" {
			cfg.Server.Port = pCfg.Server.Port
		}
	}
	if pCfg.Ethereum != nil {
		if pCfg.Ethereum.RPCURL != "" {
			cfg.Ethereum.RPCURL = pCfg.Ethereum.RPCURL
		}
	}
	if pCfg.Parser != nil {
		cfg.Parser.PollingIntervalSeconds = pCfg.Parser.PollingIntervalSeconds
		cfg.Parser.InitialScanBlockNumber = pCfg.Parser.InitialScanBlockNumber
		if cfg.Parser.PollingIntervalSeconds <= 0 {
			cfg.Parser.PollingIntervalSeconds = DefaultParserPollingInt
		}
	}

	if cfg.Server.Port == "" {
		cfg.Server.Port = DefaultServerPort
	}
	if cfg.Ethereum.RPCURL == "" {
		cfg.Ethereum.RPCURL = DefaultEthereumRPCURL
	}

	fmt.Printf("Configuration loaded from '%s'\n", loadPath)
	return cfg, nil
}
