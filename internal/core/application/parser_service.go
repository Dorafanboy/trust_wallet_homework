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

	pollCtx    context.Context
	pollCancel context.CancelFunc
	stopChan   chan struct{}
}

// Compile-time check to ensure ParserServiceImpl implements ethparser.Parser
var _ ethparser.Parser = (*ParserServiceImpl)(nil)

// Config holds configuration needed by the ParserService.
type Config struct {
	PollingIntervalSeconds int
}

// NewParserService creates a new instance of ParserServiceImpl.
func NewParserService(
	stateRepo repository.ParserStateRepository,
	addressRepo repository.MonitoredAddressRepository,
	txRepo repository.TransactionRepository,
	ethClient client.EthereumClient,
	appLogger logger.AppLogger,
	appCfg config.ApplicationServiceConfig,
) (ethparser.Parser, error) {
	// Check for nil dependencies individually, starting with logger.
	if appLogger == nil {
		// Cannot log if logger itself is nil, so just return error.
		return nil, errors.New("NewParserService: appLogger is nil")
	}
	if stateRepo == nil {
		appLogger.Error("NewParserService: stateRepo is nil") // Log using the now-confirmed non-nil appLogger
		return nil, errors.New("NewParserService: stateRepo is nil")
	}
	if addressRepo == nil {
		appLogger.Error("NewParserService: addressRepo is nil")
		return nil, errors.New("NewParserService: addressRepo is nil")
	}
	if txRepo == nil {
		appLogger.Error("NewParserService: txRepo is nil")
		return nil, errors.New("NewParserService: txRepo is nil")
	}
	if ethClient == nil {
		appLogger.Error("NewParserService: ethClient is nil")
		return nil, errors.New("NewParserService: ethClient is nil")
	}

	sInstance := &ParserServiceImpl{
		stateRepo:       stateRepo,
		addressRepo:     addressRepo,
		txRepo:          txRepo,
		ethClient:       ethClient,
		pollingInterval: time.Duration(appCfg.PollingIntervalSeconds) * time.Second,
	}

	// Simplified logic: always start from the latest network block.
	sInstance.logger.Info("Attempting to fetch latest block from network to determine starting point...")
	latestNetBlock, errNet := sInstance.ethClient.GetLatestBlockNumber(context.Background())
	if errNet != nil {
		sInstance.logger.Error("Failed to fetch latest block number from network", "error", errNet, "defaultingToBlock", 0)
		sInstance.lastKnownBlock, _ = domain.NewBlockNumber(0)
	} else {
		sInstance.lastKnownBlock = latestNetBlock
		sInstance.logger.Info("Starting scan from latest network block", "blockNumber", sInstance.lastKnownBlock.Value())
	}

	ctxForInitialSet := context.Background() // Use a new context for this operation
	if errSet := sInstance.stateRepo.SetCurrentBlock(ctxForInitialSet, sInstance.lastKnownBlock); errSet != nil {
		sInstance.logger.Error("Failed to set initial parser state in repository", "error", errSet, "blockNumber", sInstance.lastKnownBlock.Value())
	} else {
		sInstance.logger.Info("Initial parser state set in repository", "blockNumber", sInstance.lastKnownBlock.Value())
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

	// Initial scan on startup, using the already initialized s.lastKnownBlock from NewParserService.
	// s.lastKnownBlock is now set directly in NewParserService and reflects the latest network block (or 0).
	s.scanBlockRange(s.lastKnownBlock) // Pass s.lastKnownBlock directly

	for {
		select {
		case <-ticker.C:
			// Subsequent scans will use the updated s.lastKnownBlock from stateRepo after each successful scanBlockRange
			currentBlockFromState, err := s.stateRepo.GetCurrentBlock(s.pollCtx) // Use s.pollCtx here
			if err != nil {
				s.logger.Error("Failed to get current block from state before polling tick scan", "error", err)
				continue // Skip this tick if we can't get state
			}
			s.scanBlockRange(currentBlockFromState)
		case <-s.pollCtx.Done():
			s.logger.Info("Polling loop stopping due to context cancellation.")
			return
		}
	}
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

	if end > latestBlock.Value() { // This check is redundant as end is already latestBlock.Value()
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

// scanBlockRange performs a single scan iteration.
func (s *ParserServiceImpl) scanBlockRange(currentBlockFromState domain.BlockNumber) { // Accepts current block as parameter
	ctx, cancel := context.WithTimeout(s.pollCtx, s.pollingInterval-time.Second)
	defer cancel()

	logger := s.logger.With("method", "scanBlockRange")

	logger.Info("Starting scan block range iteration.")

	logger = logger.With("currentBlockToScanFrom", currentBlockFromState.Value())

	start, end, scanNeeded, err := s.getScanRange(ctx, currentBlockFromState)
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

	lastSuccessfullyProcessedBlock := currentBlockFromState.Value()

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
