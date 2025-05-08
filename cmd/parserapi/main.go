// Package main is the entry point for the Ethereum Parser API application.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"trust_wallet_homework/internal/adapters/restapi"
	"trust_wallet_homework/internal/adapters/rpc"
	"trust_wallet_homework/internal/adapters/storage/memory"
	"trust_wallet_homework/internal/config"
	"trust_wallet_homework/internal/core/application"
	"trust_wallet_homework/pkg/ethparser"
)

// main is the entry point of the application.
func main() {
	configFile := flag.String("config", "", "Path to YAML configuration file (default: config/config.yml)")
	flag.Parse()

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	logLevel := new(slog.LevelVar)
	logLevel.Set(slog.LevelDebug)
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	logger := slog.New(jsonHandler)
	slog.SetDefault(logger)

	if err := run(cfg, logger, configFile); err != nil {
		logger.Error("Application run failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Application shut down gracefully.")
}

// run initializes and starts the application components.
func run(cfg *config.Config, logger *slog.Logger, configFile *string) error {
	logMsg := "Configuration loaded successfully"
	if configFile != nil && *configFile != "" {
		logger.Info(logMsg, "configFile", *configFile)
	} else {
		logger.Info(logMsg, "configFile", config.DefaultConfigFile+" (default)")
	}

	httpClient := &http.Client{Timeout: 20 * time.Second}

	stateRepo := memory.NewInMemoryParserStateRepo()
	addrRepo := memory.NewInMemoryAddressRepo()
	txRepo := memory.NewInMemoryTransactionRepo()
	ethClient := rpc.NewEthereumNodeAdapter(cfg.Ethereum.RPCURL, httpClient)

	appCfg := application.Config{
		PollingIntervalSeconds: cfg.Parser.PollingIntervalSeconds,
		InitialScanBlockNumber: cfg.Parser.InitialScanBlockNumber,
	}
	parserService, err := application.NewParserService(stateRepo, addrRepo, txRepo, ethClient, logger, appCfg)
	if err != nil {
		return fmt.Errorf("failed to create parser service: %w", err)
	}

	var parserServiceAPI ethparser.Parser = parserService

	apiServer, err := restapi.NewServer(parserServiceAPI, logger, cfg)
	if err != nil {
		return fmt.Errorf("failed to create API server: %w", err)
	}

	logger.Info("-------------------------------------")
	logger.Info("API Server starting", "address", cfg.Server.Port)
	logger.Info("Available Endpoints:")
	logger.Info("  GET  /current_block")
	logger.Info("  POST /subscribe       (Body: {'address':'0x...'})")
	logger.Info("  GET  /transactions/{address}")
	logger.Info("-------------------------------------")

	return gracefulShutdown(logger, parserServiceAPI, apiServer)
}

// gracefulShutdown manages the startup of concurrent components and their graceful shutdown.
func gracefulShutdown(logger *slog.Logger, parserService ethparser.Parser, apiServer *restapi.Server) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errChan := make(chan error, 2)

	go func() {
		logger.Info("Starting parser service background process...")
		if errSvcStart := parserService.Start(ctx); errSvcStart != nil {
			logger.Warn("Parser service Start() returned an error", "error", errSvcStart)
		}
		<-ctx.Done()
		logger.Info("Shutdown signal received in parser goroutine, ensuring stop...")
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelShutdown()
		if errStop := parserService.Stop(shutdownCtx); errStop != nil {
			logger.Error("Parser service shutdown error", "error", errStop)
		} else {
			logger.Info("Parser service stopped (from goroutine).")
		}
	}()

	go func() {
		if errServ := apiServer.Start(); errServ != nil && !errors.Is(errServ, http.ErrServerClosed) {
			errChan <- fmt.Errorf("http server critical error: %w", errServ)
		}
	}()

	select {
	case err := <-errChan:
		logger.Error("Critical component failed to start, initiating shutdown...", "error", err)
		return err
	case <-ctx.Done():
		logger.Info("Shutting down due to OS signal...")
	}

	httpShutdownCtx, cancelHTTPShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelHTTPShutdown()

	if err := apiServer.Shutdown(httpShutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error during graceful shutdown", "error", err)
	} else {
		logger.Info("HTTP server stopped (from main shutdown sequence).")
	}

	return nil
}
