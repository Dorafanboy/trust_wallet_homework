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

// main is entry point of application.
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

	logMsg := "Configuration loaded successfully"
	if configFile != nil && *configFile != "" {
		logger.Info(logMsg, "configFile", *configFile)
	} else {
		logger.Info(logMsg, "configFile", "config/config.yml (default)")
	}

	httpClient := &http.Client{Timeout: 20 * time.Second}

	stateRepo := memory.NewInMemoryParserStateRepo()
	addrRepo := memory.NewInMemoryAddressRepo()
	txRepo := memory.NewInMemoryTransactionRepo()
	ethClient := rpc.NewEthereumNodeAdapter(cfg.EthereumRPCURL, httpClient)

	appCfg := application.Config{
		PollingIntervalSeconds: cfg.PollingIntervalSeconds,
		InitialScanBlockNumber: cfg.InitialScanBlockNumber,
	}
	parserService, err := application.NewParserService(stateRepo, addrRepo, txRepo, ethClient, logger, appCfg)
	if err != nil {
		logger.Error("Failed to create parser service", "error", err)
		os.Exit(1)
	}

	var parserServiceAPI ethparser.Parser = parserService

	apiServer := restapi.NewServer(parserServiceAPI, logger, cfg)

	logger.Info("-------------------------------------")
	logger.Info("API Server starting", "address", cfg.HTTPListenAddress)
	logger.Info("Available Endpoints:")
	logger.Info("  GET  /current_block")
	logger.Info("  POST /subscribe       (Body: {'address':'0x...'})")
	logger.Info("  GET  /transactions/{address}")
	logger.Info("-------------------------------------")

	gracefulShutdown(logger, parserServiceAPI, apiServer)

	logger.Info("Application shut down gracefully.")
}

// gracefulShutdown manages the startup of concurrent components (parser, API server).
func gracefulShutdown(logger *slog.Logger, parserService ethparser.Parser, apiServer *restapi.Server) {
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
			errChan <- fmt.Errorf("parser service shutdown error: %w", errStop)
		} else {
			logger.Info("Parser service stopped (from goroutine).")
		}
	}()

	go func() {
		if errServ := apiServer.Start(); errServ != nil && !errors.Is(errServ, http.ErrServerClosed) {
			errChan <- fmt.Errorf("http server error: %w", errServ)
		}
	}()

	select {
	case err := <-errChan:
		logger.Error("Shutting down due to error", "error", err)
	case <-ctx.Done():
		logger.Info("Shutting down due to OS signal...")
	}

	httpShutdownCtx, cancelHttpShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelHttpShutdown()

	if err := apiServer.Shutdown(httpShutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}
	logger.Info("HTTP server stopped (from main shutdown sequence).")
}
