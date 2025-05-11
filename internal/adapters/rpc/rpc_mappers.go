package rpc

import (
	"fmt"
	"log"

	"trust_wallet_homework/internal/core/domain"
	"trust_wallet_homework/internal/utils"
)

// mapRPCBlockToDomain converts the RPC DTO for a block to the domain model.
func mapRPCBlockToDomain(rpcBlock *Block) (*domain.Block, error) {
	num, err := utils.HexToInt64(rpcBlock.Number)
	if err != nil {
		return nil, fmt.Errorf("invalid block number hex '%s': %w", rpcBlock.Number, err)
	}
	domainBlockNum, err := domain.NewBlockNumber(num)
	if err != nil {
		return nil, fmt.Errorf("failed creating domain block number: %w", err)
	}

	domainBlockHash, err := domain.NewBlockHash(rpcBlock.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed creating domain block hash: %w", err)
	}

	timestamp, err := utils.HexToUint64(rpcBlock.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("invalid block timestamp hex '%s': %w", rpcBlock.Timestamp, err)
	}

	domainTxs := make([]domain.Transaction, 0, len(rpcBlock.Transactions))
	for i, rpcTx := range rpcBlock.Transactions {
		domainTx, err := mapRPCTransactionToDomain(&rpcTx, domainBlockNum, timestamp)
		if err != nil {
			log.Printf("Error mapping transaction index %d (hash: %s) in block %d: %v", i, rpcTx.Hash, num, err)
			continue
		}
		domainTxs = append(domainTxs, *domainTx)
	}

	domainBlock := domain.NewBlock(domainBlockNum, domainBlockHash, timestamp, domainTxs)
	return &domainBlock, nil
}

// mapRPCTransactionToDomain converts the RPC DTO for a transaction to the domain model.
func mapRPCTransactionToDomain(
	rpcTx *Transaction,
	blockNum domain.BlockNumber,
	blockTimestamp uint64,
) (*domain.Transaction, error) {
	hash, err := domain.NewTransactionHash(rpcTx.Hash)
	if err != nil {
		return nil, fmt.Errorf("invalid tx hash '%s': %w", rpcTx.Hash, err)
	}

	from, err := domain.NewAddress(rpcTx.From)
	if err != nil {
		return nil, fmt.Errorf("invalid tx from address '%s': %w", rpcTx.From, err)
	}

	var to domain.Address
	if rpcTx.To != nil && *rpcTx.To != "" {
		to, err = domain.NewAddress(*rpcTx.To)
		if err != nil {
			return nil, fmt.Errorf("invalid tx to address '%s': %w", *rpcTx.To, err)
		}
	}

	value, err := domain.NewWeiValue(rpcTx.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid tx value '%s': %w", rpcTx.Value, err)
	}

	domainTx := domain.NewTransaction(hash, from, to, value, blockNum, blockTimestamp)
	return &domainTx, nil
}
