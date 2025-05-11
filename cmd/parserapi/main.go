// Package main is the entry point for the Ethereum Parser API application.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	applogger "trust_wallet_homework/internal/logger"
	"trust_wallet_homework/pkg/ethparser"

	"golang.org/x/sync/errgroup"
)

const configFilePath = "config/config.yml"

// main is the entry point of the application.
func main() {
	cfg, err := config.LoadConfig(configFilePath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v\n", err)
	}

	appLogger, err := applogger.NewAppLogger(cfg.Logger)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v\n", err)
	}
	appLogger.Info("Logger initialized", "level", cfg.Logger.Level, "format", cfg.Logger.Format)

	if err := run(cfg, appLogger); err != nil {
		appLogger.Error("Application run failed", "error", err)
		os.Exit(1)
	}

	appLogger.Info("Application shut down gracefully.")
}

// run initializes and starts the application components.
func run(cfg *config.Config, logger applogger.AppLogger) error {
	baseCtx := context.Background()
	ctx, stop := signal.NotifyContext(baseCtx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	httpClient := &http.Client{Timeout: time.Duration(cfg.ETHClient.ClientTimeoutSeconds) * time.Second}

	ethNodeClient := rpc.NewEthereumNodeAdapter(cfg.ETHClient.NodeURL, httpClient)

	stateRepo := memory.NewInMemoryParserStateRepo()
	addrRepo := memory.NewInMemoryAddressRepo()
	txRepo := memory.NewInMemoryTransactionRepo()

	parserService, err := application.NewParserService(
		stateRepo,
		addrRepo,
		txRepo,
		ethNodeClient,
		logger,
		cfg.AppService,
	)
	if err != nil {
		return fmt.Errorf("failed to create parser service: %w", err)
	}

	apiServer, err := restapi.NewServer(parserService, logger, &cfg.Server)
	if err != nil {
		return fmt.Errorf("failed to create API server: %w", err)
	}

	return gracefulShutdown(ctx, logger, parserService, apiServer)
}

// gracefulShutdown manages the startup of concurrent components and their graceful shutdown.
func gracefulShutdown(
	ctx context.Context,
	logger applogger.AppLogger,
	parserService ethparser.Parser,
	apiServer *restapi.Server,
) error {
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		logger.Info("Starting parser service background process...")
		if errSvcStart := parserService.Start(gCtx); errSvcStart != nil {
			if errors.Is(errSvcStart, context.Canceled) || errors.Is(errSvcStart, context.DeadlineExceeded) {
				logger.Info("Parser service start context cancelled.")
				return nil
			}
			logger.Error("Parser service Start() failed", "error", errSvcStart)
			return fmt.Errorf("parser service failed: %w", errSvcStart)
		}
		logger.Info("Parser service Start process finished (likely due to context cancellation).")
		return nil
	})

	g.Go(func() error {
		logger.Info("Starting API server (already logged endpoints in run func)...")
		if errServ := apiServer.Start(); errServ != nil {
			if errors.Is(errServ, http.ErrServerClosed) {
				logger.Info("API server closed.")
				return nil
			}
			logger.Error("API server ListenAndServe error", "error", errServ)
			return fmt.Errorf("http server critical error: %w", errServ)
		}
		logger.Info("API server Start process finished (likely due to http.ErrServerClosed or other termination).")
		return nil
	})

	logger.Info("Application services started via errgroup. Waiting for OS signal or critical error...")

	serverErr := g.Wait()

	if serverErr != nil {
		logger.Error("A service failed or OS signal received, initiating shutdown...", "cause_error", serverErr)
	} else {
		logger.Info("OS signal received or services completed, initiating graceful shutdown...")
	}

	logger.Info("Attempting to stop parser service...")
	parserShutdownCtx, cancelParserShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelParserShutdown()
	if err := parserService.Stop(parserShutdownCtx); err != nil {
		logger.Error("Parser service graceful shutdown error", "error", err)
	} else {
		logger.Info("Parser service stopped successfully.")
	}

	logger.Info("Attempting to stop API server...")
	httpShutdownCtx, cancelHTTPShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelHTTPShutdown()
	if err := apiServer.Shutdown(httpShutdownCtx); err != nil {
		logger.Error("HTTP server graceful shutdown error", "error", err)
	} else {
		logger.Info("HTTP server stopped successfully.")
	}

	if errors.Is(serverErr, context.Canceled) {
		return nil
	}
	return serverErr
}
