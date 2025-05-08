// Package client defines interfaces for external service clients, such as an Ethereum node client.
//
//go:generate mockgen -source=$GOFILE -destination=../../mocks/mock_$GOPACKAGE/mock_$GOFILE -package=mock_$GOPACKAGE
package client

import (
	"context"

	"trust_wallet_homework/internal/core/domain"
)

// EthereumClient defines the interface for interacting with an Ethereum node.
type EthereumClient interface {
	// GetLatestBlockNumber fetches the number of the most recent block in the blockchain.
	GetLatestBlockNumber(ctx context.Context) (domain.BlockNumber, error)

	// GetBlockWithTransactions fetches a block by its number, including all transaction details.
	GetBlockWithTransactions(ctx context.Context, blockNumber domain.BlockNumber) (*domain.Block, error)
}
