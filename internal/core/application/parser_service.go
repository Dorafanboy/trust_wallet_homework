// Package application contains the core application service logic for the Ethereum parser.
package application

import (
	"context"
	"errors"
	"fmt"
	"time"

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

	pollingInterval                   time.Duration
	initialScanBlockNumberConfigValue int64
	initialScanBlock                  domain.BlockNumber

	pollCtx    context.Context
	pollCancel context.CancelFunc
	stopChan   chan struct{}
}

// Compile-time check to ensure ParserServiceImpl implements ethparser.Parser
var _ ethparser.Parser = (*ParserServiceImpl)(nil)

// Config holds configuration needed by the ParserService.
type Config struct {
	PollingIntervalSeconds int
	InitialScanBlockNumber int64
}

// NewParserService creates a new instance of ParserServiceImpl.
func NewParserService(
	stateRepo repository.ParserStateRepository,
	addressRepo repository.MonitoredAddressRepository,
	txRepo repository.TransactionRepository,
	ethClient client.EthereumClient,
	appLogger logger.AppLogger,
	cfg Config,
) (*ParserServiceImpl, error) {
	if appLogger == nil {
		return nil, fmt.Errorf("logger cannot be nil for ParserService")
	}

	if cfg.PollingIntervalSeconds <= 0 {
		cfg.PollingIntervalSeconds = 15
	}

	var initialBlockForState domain.BlockNumber
	if cfg.InitialScanBlockNumber >= 0 {
		block, err := domain.NewBlockNumber(cfg.InitialScanBlockNumber)
		if err != nil {
			appLogger.Warn("Invalid non-negative InitialScanBlockNumber",
				"configValue", cfg.InitialScanBlockNumber,
				"error", err)
			initialBlockForState, _ = domain.NewBlockNumber(0)
		} else {
			initialBlockForState = block
		}
	} else {
		initialBlockForState, _ = domain.NewBlockNumber(0)
	}

	s := &ParserServiceImpl{
		stateRepo:                         stateRepo,
		addressRepo:                       addressRepo,
		txRepo:                            txRepo,
		ethClient:                         ethClient,
		logger:                            appLogger,
		pollingInterval:                   time.Duration(cfg.PollingIntervalSeconds) * time.Second,
		initialScanBlockNumberConfigValue: cfg.InitialScanBlockNumber,
		initialScanBlock:                  initialBlockForState,
	}

	return s, nil
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

// mapDomainToAPITransaction converts an internal domain Transaction to the public API Transaction DTO.
func mapDomainToAPITransaction(domainTx domain.Transaction) ethparser.Transaction {
	return ethparser.Transaction{
		Hash:        domainTx.Hash.String(),
		From:        domainTx.From.String(),
		To:          domainTx.To.String(),
		Value:       domainTx.Value.String(),
		BlockNumber: domainTx.BlockNumber.Value(),
		Timestamp:   domainTx.Timestamp,
	}
}

// Start initiates the background blockchain polling process.
func (s *ParserServiceImpl) Start(_ context.Context) (err error) {
	if s.pollCancel != nil {
		if s.pollCtx.Err() == nil {
			s.logger.Info("Parser service is already running.")
			return fmt.Errorf("service already running")
		}
	}

	s.pollCtx, s.pollCancel = context.WithCancel(context.Background())
	s.stopChan = make(chan struct{})

	go s.pollBlocks()
	s.logger.Info("Parser service started polling...")
	return nil
}

// Stop signals the background polling process to shut down gracefully and waits for it to complete.
func (s *ParserServiceImpl) Stop(ctx context.Context) (err error) {
	if s.pollCancel == nil || s.pollCtx.Err() != nil {
		s.logger.Info("Parser service is not running or already stopped.")
		return nil
	}

	s.logger.Info("Stopping parser service...")
	s.pollCancel()

	select {
	case <-s.stopChan:
		s.logger.Info("Parser service stopped gracefully.")
		return nil
	case <-ctx.Done():
		s.logger.Error("Parser service stop timed out.", "error", ctx.Err())
		return ctx.Err()
	}
}

// pollBlocks is the main background loop for scanning the blockchain.
func (s *ParserServiceImpl) pollBlocks() {
	defer close(s.stopChan)
	ticker := time.NewTicker(s.pollingInterval)
	defer ticker.Stop()

	s.logger.Info("Polling loop started.")

	s.scanBlockRange()

	for {
		select {
		case <-ticker.C:
			s.scanBlockRange()
		case <-s.pollCtx.Done():
			s.logger.Info("Polling loop stopping due to context cancellation.")
			return
		}
	}
}

// initializeStateIfRequired checks if the parser state is initialized in the repository.
func (s *ParserServiceImpl) initializeStateIfRequired(ctx context.Context) (domain.BlockNumber, error) {
	currentParsedBlock, stateErr := s.stateRepo.GetCurrentBlock(ctx)

	if stateErr != nil && errors.Is(stateErr, repository.ErrStateNotInitialized) {
		s.logger.Info("State not initialized", "error", stateErr)

		if s.initialScanBlockNumberConfigValue == -1 {
			s.logger.Info("InitialScanBlockNumber is -1, fetching latest block to determine starting point...")
			latestBlockNum, latestErr := s.ethClient.GetLatestBlockNumber(ctx)
			if latestErr != nil {
				s.logger.Error("Failed to get latest block number for initial state", "error", latestErr)
				return domain.BlockNumber{}, fmt.Errorf("failed to get latest block number for initial state: %w", latestErr)
			}
			currentParsedBlock = latestBlockNum
			s.logger.Info("Initial state to be set to latest block", "blockNumber", currentParsedBlock.Value())
		} else {
			currentParsedBlock = s.initialScanBlock
			s.logger.Info("Initial state to be set to configured block", "blockNumber", currentParsedBlock.Value())
		}
		if err := s.stateRepo.SetCurrentBlock(ctx, currentParsedBlock); err != nil {
			s.logger.Error("Failed to set initial block state", "block", currentParsedBlock.Value(), "error", err)
			return domain.BlockNumber{}, fmt.Errorf(
				"failed to set initial block state to %d: %w",
				currentParsedBlock.Value(),
				err,
			)
		}
		s.logger.Info("Initial state set to block", "blockNumber", currentParsedBlock.Value())
		return currentParsedBlock, nil
	} else if stateErr != nil {
		s.logger.Error("Error getting current block from state", "error", stateErr)
		return domain.BlockNumber{}, fmt.Errorf("error getting current block from state: %w", stateErr)
	}

	return currentParsedBlock, nil
}

// getScanRange determines the block range to scan in the current iteration.
func (s *ParserServiceImpl) getScanRange(
	ctx context.Context,
	currentParsedBlock domain.BlockNumber,
) (start, end int64, scanNeeded bool, err error) {
	logger := s.logger.With("currentParsedBlock", currentParsedBlock.Value())
	latestBlock, fetchErr := s.ethClient.GetLatestBlockNumber(ctx)
	if fetchErr != nil {
		logger.Error("Error getting latest block number", "error", fetchErr)
		return 0, 0, false, fmt.Errorf("error getting latest block number: %w", fetchErr)
	}

	start = currentParsedBlock.Value() + 1
	end = latestBlock.Value()

	if end > latestBlock.Value() {
		end = latestBlock.Value()
	}

	if start > end {
		logger.Info("No new blocks to scan", "latestBlockOnNode", latestBlock.Value())
		return 0, 0, false, nil
	}

	return start, end, true, nil
}

// processBlock fetches a single block, finds relevant transactions based on monitored addresses,
func (s *ParserServiceImpl) processBlock(
	ctx context.Context,
	blockNum domain.BlockNumber,
	monitoredAddresses map[string]struct{},
) error {
	logger := s.logger.With("blockNumber", blockNum.Value())
	logger.Debug("Processing block")

	block, err := s.ethClient.GetBlockWithTransactions(ctx, blockNum)
	if err != nil {
		logger.Error("Failed to get block with transactions", "error", err)
		return fmt.Errorf("failed to get block %d: %w", blockNum.Value(), err)
	}

	if block == nil {
		logger.Warn("Received nil block, skipping")
		return nil
	}

	logger = logger.With("blockHash", block.Hash.String(), "txCount", len(block.Transactions))
	foundTxs := 0
	for _, tx := range block.Transactions {
		storeTx := false
		if _, ok := monitoredAddresses[tx.From.String()]; ok {
			storeTx = true
		}
		if !tx.To.IsZero() {
			if _, ok := monitoredAddresses[tx.To.String()]; ok {
				storeTx = true
			}
		}

		if storeTx {
			if err := s.txRepo.Store(ctx, tx); err != nil {
				logger.Error("Failed to store transaction", "txHash", tx.Hash.String(), "error", err)
			} else {
				foundTxs++
			}
		}
	}
	if foundTxs > 0 {
		logger.Info("Stored transactions from block", "storedTxCount", foundTxs)
	}

	return nil
}

// scanBlockRange performs a single scan iteration. It initializes state if needed,
func (s *ParserServiceImpl) scanBlockRange() {
	ctx, cancel := context.WithTimeout(s.pollCtx, s.pollingInterval-time.Second)
	defer cancel()

	logger := s.logger.With("method", "scanBlockRange")

	logger.Info("Starting scan block range iteration.")

	currentParsedBlock, err := s.initializeStateIfRequired(ctx)
	if err != nil {
		logger.Error("Failed to initialize state or get current block", "error", err)
		return
	}

	logger = logger.With("currentParsedBlockFromState", currentParsedBlock.Value())

	start, end, scanNeeded, err := s.getScanRange(ctx, currentParsedBlock)
	if err != nil {
		logger.Error("Failed to determine scan range", "error", err)
		return
	}

	if !scanNeeded {
		logger.Info("Scan not needed in this iteration.")
		return
	}

	logger.Info("Scanning blocks", "from", start, "to", end)

	monitoredAddressList, err := s.addressRepo.FindAll(ctx)
	if err != nil {
		logger.Error("Failed to get monitored addresses", "error", err)
		return
	}

	monitoredAddressesMap := make(map[string]struct{}, len(monitoredAddressList))
	for _, addr := range monitoredAddressList {
		monitoredAddressesMap[addr.String()] = struct{}{}
	}

	if len(monitoredAddressesMap) == 0 {
		logger.Info("No addresses are currently subscribed for monitoring. Skipping transaction processing until subscribed.")
	}

	lastSuccessfullyProcessedBlock := currentParsedBlock.Value()

	for i := start; i <= end; i++ {
		select {
		case <-ctx.Done():
			logger.Warn("Scan block range context done during block processing loop",
				"lastProcessed", lastSuccessfullyProcessedBlock,
				"error", ctx.Err())
			finalBlockNum, _ := domain.NewBlockNumber(lastSuccessfullyProcessedBlock)
			if updateErr := s.stateRepo.SetCurrentBlock(s.pollCtx, finalBlockNum); updateErr != nil {
				logger.Error("Failed to update current block state on scan interruption",
					"blockNumber", lastSuccessfullyProcessedBlock,
					"error", updateErr)
			}
			return
		default:
			blockNumToProcess, _ := domain.NewBlockNumber(i)
			if err := s.processBlock(ctx, blockNumToProcess, monitoredAddressesMap); err != nil {
				logger.Error("Failed to process block, stopping current scan iteration", "blockNumber", i, "error", err)
				finalBlockNum, _ := domain.NewBlockNumber(lastSuccessfullyProcessedBlock)
				if updateErr := s.stateRepo.SetCurrentBlock(s.pollCtx, finalBlockNum); updateErr != nil {
					logger.Error("Failed to update current block state after processing error",
						"blockNumber", lastSuccessfullyProcessedBlock,
						"error", updateErr)
				}
				return
			}
			lastSuccessfullyProcessedBlock = i
		}
	}

	finalBlockNum, _ := domain.NewBlockNumber(lastSuccessfullyProcessedBlock)
	if err := s.stateRepo.SetCurrentBlock(s.pollCtx, finalBlockNum); err != nil {
		logger.Error("Failed to update current block state after scan range completion",
			"blockNumber", lastSuccessfullyProcessedBlock,
			"error", err)
	} else {
		logger.Info("Successfully scanned and updated current block", "processedUpToBlock", lastSuccessfullyProcessedBlock)
	}
}
