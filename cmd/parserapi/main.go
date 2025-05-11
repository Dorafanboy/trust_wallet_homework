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
			logger.Error("Parser service Start() call returned an error", "error", errSvcStart)
			return fmt.Errorf("parser service Start() failed: %w", errSvcStart)
		}
		<-gCtx.Done()
		logger.Info("Parser service Start goroutine: context cancelled. Waiting for parser to stop...")
		return nil
	})

	g.Go(func() error {
		logger.Info("Starting API server...")
		serverErrChan := make(chan error, 1)
		go func() {
			logger.Info("API server ListenAndServe starting...")
			if errServ := apiServer.Start(); errServ != nil && !errors.Is(errServ, http.ErrServerClosed) {
				serverErrChan <- fmt.Errorf("http server critical error: %w", errServ)
			} else {
				close(serverErrChan)
			}
		}()

		select {
		case <-gCtx.Done():
			logger.Info("API server: context cancelled, initiating shutdown...")
			shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancelShutdown()
			if err := apiServer.Shutdown(shutdownCtx); err != nil {
				logger.Error("API server graceful shutdown error", "error", err)
				return fmt.Errorf("api server shutdown failed: %w", err)
			}
			logger.Info("API server shut down gracefully due to context cancellation.")
			if errFromStart, ok := <-serverErrChan; ok && errFromStart != nil {
				logger.Error("API server Start() returned an unexpected error", "error", errFromStart)
				return errFromStart
			}
			return nil
		case err, ok := <-serverErrChan:
			if !ok {
				logger.Info("API server Start() goroutine completed (channel closed).")
				return nil
			}
			logger.Error("API server ListenAndServe failed", "error", err)
			return err
		}
	})

	waitErr := g.Wait()

	if waitErr != nil {
		if errors.Is(waitErr, context.Canceled) {
			logger.Info("Errgroup context cancelled (likely SIGINT/SIGTERM), proceeding with final cleanup.")
		} else {
			logger.Error("A service within errgroup failed", "error", waitErr)
		}
	}

	parserShutdownCtx, cancelParserShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelParserShutdown()
	if err := parserService.Stop(parserShutdownCtx); err != nil {
		logger.Error("Parser service graceful shutdown error (post g.Wait)", "error", err)
		if !errors.Is(waitErr, context.Canceled) {
			if waitErr == nil {
				waitErr = fmt.Errorf("parser service stop failed: %w", err)
			} else {
				waitErr = fmt.Errorf("parser service stop failed (%w) after initial error (%w)", err, waitErr)
			}
		}
	}

	if errors.Is(waitErr, context.Canceled) {
		return nil
	}
	return waitErr
}
