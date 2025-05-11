// Package application contains the core application service logic for the Ethereum parser.
package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"trust_wallet_homework/internal/config"
	"trust_wallet_homework/internal/core/domain"
	"trust_wallet_homework/internal/core/domain/client"
	"trust_wallet_homework/internal/core/domain/repository"
	"trust_wallet_homework/internal/logger"
	"trust_wallet_homework/pkg/ethparser"
)

// ParserServiceImpl implements the ethparser.Parser interface and contains the core application logic.
type ParserServiceImpl struct {
	stateRepo   repository.ParserStateRepository
	addressRepo repository.MonitoredAddressRepository
	txRepo      repository.TransactionRepository
	ethClient   client.EthereumClient
	logger      logger.AppLogger

	pollingInterval time.Duration
	lastKnownBlock  domain.BlockNumber

	pollCtx  context.Context
	stopChan chan struct{}
}

// Compile-time check to ensure ParserServiceImpl implements ethparser.Parser
var _ ethparser.Parser = (*ParserServiceImpl)(nil)

// NewParserService creates a new instance of ParserServiceImpl.
func NewParserService(
	stateRepo repository.ParserStateRepository,
	addressRepo repository.MonitoredAddressRepository,
	txRepo repository.TransactionRepository,
	ethClient client.EthereumClient,
	appLogger logger.AppLogger,
	appCfg config.ApplicationServiceConfig,
) (*ParserServiceImpl, error) {
	if appLogger == nil {
		return nil, errors.New("NewParserService: appLogger is nil")
	}
	if stateRepo == nil {
		return nil, errors.New("NewParserService: stateRepo is nil")
	}
	if addressRepo == nil {
		return nil, errors.New("NewParserService: addressRepo is nil")
	}
	if txRepo == nil {
		return nil, errors.New("NewParserService: txRepo is nil")
	}
	if ethClient == nil {
		return nil, errors.New("NewParserService: ethClient is nil")
	}

	sInstance := &ParserServiceImpl{
		stateRepo:       stateRepo,
		addressRepo:     addressRepo,
		txRepo:          txRepo,
		ethClient:       ethClient,
		logger:          appLogger,
		pollingInterval: time.Duration(appCfg.PollingIntervalSeconds) * time.Second,
	}

	return sInstance, nil
}

// GetCurrentBlock returns the number of the last successfully parsed block.
func (s *ParserServiceImpl) GetCurrentBlock(ctx context.Context) (blockNumber int64, err error) {
	domainBlockNumber, err := s.stateRepo.GetCurrentBlock(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get current block from state: %w", err)
	}
	return domainBlockNumber.Value(), nil
}

// Subscribe adds a new address to be monitored by the parser.
func (s *ParserServiceImpl) Subscribe(ctx context.Context, addressString string) (err error) {
	address, err := domain.NewAddress(addressString)
	if err != nil {
		return fmt.Errorf("address validation failed: %w", err)
	}

	loggerWithAddress := s.logger.With("address", address.String())
	if err := s.addressRepo.Add(ctx, address); err != nil {
		loggerWithAddress.Error("Failed to subscribe address in repository", "error", err)
		return fmt.Errorf("failed to subscribe address in repository: %w", err)
	}

	s.logger.Info("Successfully subscribed address", "address", address.String())
	return nil
}

// GetTransactions retrieves transactions associated with a given monitored address.
func (s *ParserServiceImpl) GetTransactions(
	ctx context.Context,
	addressString string,
) ([]ethparser.Transaction, error) {
	address, err := domain.NewAddress(addressString)
	if err != nil {
		return nil, fmt.Errorf("address validation failed: %w", err)
	}

	loggerWithAddress := s.logger.With("address", address.String())
	domainTxs, err := s.txRepo.FindByAddress(ctx, address)
	if err != nil {
		loggerWithAddress.Error("Error fetching transactions for address", "error", err)
		return nil, fmt.Errorf("failed to get transactions from repository: %w", err)
	}

	apiTxs := make([]ethparser.Transaction, 0, len(domainTxs))
	for _, domainTx := range domainTxs {
		apiTxs = append(apiTxs, mapDomainToAPITransaction(domainTx))
	}

	return apiTxs, nil
}

// Start initiates the background blockchain polling process.
func (s *ParserServiceImpl) Start(ctx context.Context) (err error) {
	s.logger.Info("Attempting to fetch latest block from network to determine starting point...")
	latestNetBlock, errNet := s.ethClient.GetLatestBlockNumber(ctx)
	if errNet != nil {
		s.logger.Error("Failed to fetch latest block number from network", "error", errNet, "defaultingToBlock", 0)
		s.lastKnownBlock, _ = domain.NewBlockNumber(0)
	} else {
		s.lastKnownBlock = latestNetBlock
		s.logger.Info("Starting scan from latest network block", "blockNumber", s.lastKnownBlock.Value())
	}

	if errSet := s.stateRepo.SetCurrentBlock(ctx, s.lastKnownBlock); errSet != nil {
		s.logger.Error("Failed to set initial parser state in repository",
			"error", errSet,
			"blockNumber", s.lastKnownBlock.Value())
	} else {
		s.logger.Info("Initial parser state set in repository", "blockNumber", s.lastKnownBlock.Value())
	}

	if s.pollCtx != nil && s.pollCtx.Err() == nil {
		s.logger.Info("Parser service is already running or was not properly stopped.")
		return fmt.Errorf("service already running or not properly stopped")
	}

	s.pollCtx = ctx
	s.stopChan = make(chan struct{})

	go s.pollBlocks()
	s.logger.Info("Parser service started polling...")
	return nil
}

// Stop signals the background polling process to shut down gracefully and waits for it to complete.
func (s *ParserServiceImpl) Stop(ctx context.Context) (err error) {
	if s.pollCtx == nil {
		s.logger.Info("Parser service was not started or already stopped.")
		return nil
	}

	if s.pollCtx.Err() != nil {
		s.logger.Info("Parser service polling context already done.")
		if s.stopChan != nil {
			s.logger.Info("Waiting for pollBlocks to confirm stop due to already done context...")
			select {
			case <-s.stopChan:
				s.logger.Info("pollBlocks confirmed stop.")
			case <-ctx.Done():
				s.logger.Error("Parser service stop timed out while waiting for pollBlocks confirmation.", "error", ctx.Err())
				return ctx.Err()
			}
		}
		return nil
	}

	s.logger.Info("Stopping parser service (external Stop call)...")
	select {
	case <-s.stopChan:
		s.logger.Info("Parser service stopped gracefully (via external Stop call).")
		return nil
	case <-ctx.Done():
		s.logger.Error("Parser service stop timed out (via external Stop call).", "error", ctx.Err())
		return ctx.Err()
	}
}
