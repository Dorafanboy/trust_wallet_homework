// Package ethparser defines the public API contracts for the Ethereum parser service.
package ethparser

import (
	"context"
)

// Transaction represents the data structure for a transaction returned by the API.
type Transaction struct {
	Hash        string `json:"hash"`
	From        string `json:"from"`
	To          string `json:"to"`
	Value       string `json:"value"`
	BlockNumber int64  `json:"blockNumber"`
	Timestamp   uint64 `json:"timestamp"`
}

// SubscribeRequestDTO represents the expected JSON body for a subscription request.
type SubscribeRequestDTO struct {
	Address string `json:"address" validate:"required,eth_addr"`
}

// Parser defines the public interface for the Ethereum blockchain parser service.
type Parser interface {
	// GetCurrentBlock returns the number of the last block that was successfully processed.
	GetCurrentBlock(ctx context.Context) (blockNumber int64, err error)

	// Subscribe adds an Ethereum address (in string format) to the list of monitored addresses.
	Subscribe(ctx context.Context, address string) (err error)

	// GetTransactions retrieves all stored transactions (both inbound and outbound)
	GetTransactions(ctx context.Context, address string) (transactions []Transaction, err error)

	// Start initiates the background process of polling for new blocks and parsing transactions.
	Start(ctx context.Context) (err error)

	// Stop gracefully shuts down the background polling process.
	Stop(ctx context.Context) (err error)
}
