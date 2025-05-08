// Package repository defines interfaces for data storage and retrieval operations.
//
//go:generate mockgen -source=$GOFILE -destination=../../mocks/mock_$GOPACKAGE/mock_$GOFILE -package=mock_$GOPACKAGE
package repository

import (
	"context"
	"errors"

	"trust_wallet_homework/internal/core/domain"
)

// ErrStateNotInitialized indicates that the parser state (e.g., the last scanned block)
var ErrStateNotInitialized = errors.New("parser state not initialized")

// ParserStateRepository defines the interface for accessing and modifying.
type ParserStateRepository interface {
	// GetCurrentBlock retrieves the number of the last block that was successfully processed.
	GetCurrentBlock(ctx context.Context) (domain.BlockNumber, error)

	// SetCurrentBlock updates the number of the last successfully processed block.
	SetCurrentBlock(ctx context.Context, blockNumber domain.BlockNumber) error
}
