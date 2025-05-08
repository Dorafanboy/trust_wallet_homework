// Package application contains the core application service logic for the Ethereum parser.
package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"trust_wallet_homework/internal/core/domain"
	"trust_wallet_homework/internal/core/domain/client"
	"trust_wallet_homework/internal/core/domain/repository"
	"trust_wallet_homework/pkg/ethparser"
)

// ParserServiceImpl implements the ethparser.Parser interface and contains the core application logic.
type ParserServiceImpl struct {
	stateRepo   repository.ParserStateRepository
	addressRepo repository.MonitoredAddressRepository
	txRepo      repository.TransactionRepository
	ethClient   client.EthereumClient
	logger      *slog.Logger

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
	logger *slog.Logger,
	cfg Config,
) (*ParserServiceImpl, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil for ParserService")
	}

	if cfg.PollingIntervalSeconds <= 0 {
		cfg.PollingIntervalSeconds = 15
	}

	var initialBlockForState domain.BlockNumber
	if cfg.InitialScanBlockNumber >= 0 {
		block, err := domain.NewBlockNumber(cfg.InitialScanBlockNumber)
		if err != nil {
			logger.Warn("Invalid non-negative InitialScanBlockNumber", "configValue", cfg.InitialScanBlockNumber, "error", err)
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
		logger:                            logger,
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

	if err := s.addressRepo.Add(ctx, address); err != nil {
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

	domainTxs, err := s.txRepo.FindByAddress(ctx, address)
	if err != nil {
		s.logger.Error("Error fetching transactions for address", "address", address.String(), "error", err)
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
		s.logger.Info("Parser service stop timed out.")
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
				return domain.BlockNumber{}, fmt.Errorf("failed to get latest block number for initial state: %w", latestErr)
			}
			currentParsedBlock = latestBlockNum
			s.logger.Info("Initial state to be set to latest block", "blockNumber", currentParsedBlock.Value())
		} else {
			currentParsedBlock = s.initialScanBlock
			s.logger.Info("Initial state to be set to configured block", "blockNumber", currentParsedBlock.Value())
		}
		if err := s.stateRepo.SetCurrentBlock(ctx, currentParsedBlock); err != nil {
			return domain.BlockNumber{}, fmt.Errorf(
				"failed to set initial block state to %d: %w",
				currentParsedBlock.Value(),
				err,
			)
		}
		s.logger.Info("Initial state set to block", "blockNumber", currentParsedBlock.Value())
		return currentParsedBlock, nil
	} else if stateErr != nil {
		return domain.BlockNumber{}, fmt.Errorf("error getting current block from state: %w", stateErr)
	}

	return currentParsedBlock, nil
}

// getScanRange determines the block range to scan in the current iteration.
func (s *ParserServiceImpl) getScanRange(
	ctx context.Context,
	currentParsedBlock domain.BlockNumber,
) (start, end int64, scanNeeded bool, err error) {
	latestBlock, fetchErr := s.ethClient.GetLatestBlockNumber(ctx)
	if fetchErr != nil {
		return 0, 0, false, fmt.Errorf("error getting latest block number: %w", fetchErr)
	}

	start = currentParsedBlock.Value() + 1
	end = latestBlock.Value()

	if start > end {
		return start, end, false, nil
	}

	return start, end, true, nil
}

// processBlock fetches a single block, finds relevant transactions based on monitored addresses,
func (s *ParserServiceImpl) processBlock(
	ctx context.Context,
	blockNum domain.BlockNumber,
	monitoredAddresses map[string]struct{},
) error {
	s.logger.Info("Fetching block", "blockNumber", blockNum.Value())

	block, fetchErr := s.ethClient.GetBlockWithTransactions(ctx, blockNum)
	if fetchErr != nil {
		s.logger.Error("Error getting block", "blockNumber", blockNum.Value(), "error", fetchErr)
		return fmt.Errorf("failed to get block %d: %w", blockNum.Value(), fetchErr)
	}

	if block == nil {
		s.logger.Warn("Block not found (nil response), stopping scan range.", "blockNumber", blockNum.Value())
		return fmt.Errorf("block %d not found (nil response from RPC)", blockNum.Value())
	}

	foundRelevantTx := false
	if len(monitoredAddresses) > 0 {
		for _, tx := range block.Transactions {
			_, fromMatch := monitoredAddresses[tx.From.String()]
			_, toMatch := monitoredAddresses[tx.To.String()]

			if fromMatch || toMatch {
				foundRelevantTx = true
				s.logger.Info("Found relevant transaction", "transactionHash", tx.Hash.String(), "blockNumber", blockNum.Value())
				if err := s.txRepo.Store(ctx, tx); err != nil {
					s.logger.Error("Error storing transaction",
						"transactionHash", tx.Hash.String(),
						"blockNumber", blockNum.Value(),
						"error", err,
					)
				}
			}
		}
	}

	if err := s.stateRepo.SetCurrentBlock(ctx, blockNum); err != nil {
		s.logger.Error("Error updating current block state to", "blockNumber", blockNum.Value(), "error", err)
		return fmt.Errorf("failed to update block state for block %d: %w", blockNum.Value(), err)
	}

	if foundRelevantTx {
		s.logger.Info("Finished processing block with relevant transactions", "blockNumber", blockNum.Value())
	}

	return nil
}

// scanBlockRange performs a single scan iteration. It initializes state if needed,
func (s *ParserServiceImpl) scanBlockRange() {
	ctx := s.pollCtx

	currentParsedBlock, err := s.initializeStateIfRequired(ctx)
	if err != nil {
		s.logger.Error("Failed to initialize or get parser state", "error", err)
		return
	}

	startScan, endScan, scanNeeded, err := s.getScanRange(ctx, currentParsedBlock)
	if err != nil {
		s.logger.Error("Failed to determine scan range", "error", err)
		return
	}

	if !scanNeeded {
		return
	}

	s.logger.Info("Scanning blocks from", "startScan", startScan, "endScan", endScan)

	monitoredAddressesDomain, err := s.addressRepo.FindAll(ctx)
	if err != nil {
		s.logger.Error("Error getting monitored addresses", "error", err)
		return
	}

	addressesMap := make(map[string]struct{}, len(monitoredAddressesDomain))
	if len(monitoredAddressesDomain) > 0 {
		for _, addr := range monitoredAddressesDomain {
			addressesMap[addr.String()] = struct{}{}
		}
	} else {
		s.logger.Info("No addresses subscribed, skipping transaction processing, will update state to latest block.")
		latestBlockNum, err := domain.NewBlockNumber(endScan)
		if err == nil {
			if errState := s.stateRepo.SetCurrentBlock(ctx, latestBlockNum); errState != nil {
				s.logger.Error("Error updating current block state to latest when no addresses monitored",
					"blockNumber", endScan,
					"error", errState,
				)
			}
		} else {
			s.logger.Error("Error creating block number from endScan", "endScan", endScan, "error", err)
		}
		return
	}

	var processingError error
	var blockNumInt int64

	for blockNumInt = startScan; blockNumInt <= endScan; blockNumInt++ {
		select {
		case <-ctx.Done():
			s.logger.Info("Block scanning interrupted by context cancellation.")
			return
		default:
		}

		blockNum, errBlock := domain.NewBlockNumber(blockNumInt)
		if errBlock != nil {
			s.logger.Error("Invalid block number encountered during scan", "blockNumInt", blockNumInt, "error", errBlock)
			processingError = fmt.Errorf("invalid block number %d: %w", blockNumInt, errBlock)
			break
		}

		if err := s.processBlock(ctx, blockNum, addressesMap); err != nil {
			processingError = err
			break
		}
	}

	if processingError != nil {
		processedSuccessfullyUpTo := blockNumInt - 1
		if blockNumInt == startScan {
			processedSuccessfullyUpTo = startScan - 1
		}
		if processedSuccessfullyUpTo < 0 {
			processedSuccessfullyUpTo = 0
		}

		s.logger.Error("Finished scanning block range with error",
			"startScan", startScan,
			"intendedEndScan", endScan,
			"processedSuccessfullyUpTo", processedSuccessfullyUpTo,
			"failedAtBlock", blockNumInt,
			"error", processingError)
	} else {
		s.logger.Info("Finished scanning block range successfully", "startScan", startScan, "endScan", endScan)
	}
}
