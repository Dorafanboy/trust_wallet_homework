package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"trust_wallet_homework/internal/core/domain"
)

// pollBlocks is the main background loop for scanning the blockchain.
func (s *ParserServiceImpl) pollBlocks() {
	defer close(s.stopChan)
	ticker := time.NewTicker(s.pollingInterval)
	defer ticker.Stop()

	s.logger.Info("Polling loop started.")

	s.scanBlockRange(s.lastKnownBlock)

	for {
		select {
		case <-ticker.C:
			currentBlockFromState, err := s.stateRepo.GetCurrentBlock(s.pollCtx)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					s.logger.Info("Polling loop: context cancelled while getting current block from state.", "error", err)
					return
				}
				s.logger.Error("Failed to get current block from state before polling tick scan", "error", err)
				continue
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
		if errors.Is(fetchErr, context.Canceled) || errors.Is(fetchErr, context.DeadlineExceeded) {
			logger.Info("Context cancelled while fetching latest block number in getScanRange.", "error", fetchErr)
			return 0, 0, false, fetchErr
		}
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
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			logger.Info("Context cancelled while getting block with transactions.", "error", err)
			return err
		}
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
		select {
		case <-ctx.Done():
			logger.Info("Context cancelled during transaction processing loop.", "error", ctx.Err())
			return ctx.Err()
		default:
		}

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
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					logger.Info("Context cancelled while storing transaction.", "error", err)
					return err
				}
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
func (s *ParserServiceImpl) scanBlockRange(currentBlockFromState domain.BlockNumber) {
	scanTimeout := s.pollingInterval - time.Second
	if scanTimeout <= 0 {
		scanTimeout = time.Millisecond * 500
	}
	scanCtx, cancelScan := context.WithTimeout(s.pollCtx, scanTimeout)
	defer cancelScan()

	logger := s.logger.With("method", "scanBlockRange")

	logger.Info("Starting scan block range iteration.")

	logger = logger.With("currentBlockToScanFrom", currentBlockFromState.Value())

	start, end, scanNeeded, err := s.getScanRange(scanCtx, currentBlockFromState)
	if err != nil {
		if !(errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
			logger.Error("Failed to determine scan range", "error", err)
		}
		return
	}

	if !scanNeeded {
		logger.Info("Scan not needed in this iteration.")
		return
	}

	logger.Info("Scanning blocks", "from", start, "to", end)

	monitoredAddressList, err := s.addressRepo.FindAll(scanCtx)
	if err != nil {
		if !(errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
			logger.Error("Failed to get monitored addresses", "error", err)
		}
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
		case <-scanCtx.Done():
			logger.Warn("Scan block range context done during block processing loop",
				"lastProcessed", lastSuccessfullyProcessedBlock,
				"error", scanCtx.Err())
			finalBlockNum, _ := domain.NewBlockNumber(lastSuccessfullyProcessedBlock)
			if updateErr := s.stateRepo.SetCurrentBlock(s.pollCtx, finalBlockNum); updateErr != nil {
				logger.Error("Failed to update current block state on scan interruption",
					"blockNumber", lastSuccessfullyProcessedBlock,
					"error", updateErr)
			}
			return
		default:
			blockNumToProcess, _ := domain.NewBlockNumber(i)
			if err := s.processBlock(scanCtx, blockNumToProcess, monitoredAddressesMap); err != nil {
				if !(errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
					logger.Error("Failed to process block, stopping current scan iteration", "blockNumber", i, "error", err)
				}
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
