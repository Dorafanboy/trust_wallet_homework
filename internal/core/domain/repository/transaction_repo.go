// Package repository defines interfaces for data storage and retrieval operations.
//
//go:generate mockgen -source=$GOFILE -destination=../../mocks/mock_$GOPACKAGE/mock_$GOFILE -package=mock_$GOPACKAGE
package repository

import (
	"context"

	"trust_wallet_homework/internal/core/domain"
)

// TransactionRepository defines the interface for storing and retrieving.
type TransactionRepository interface {
	// Store saves a transaction to the persistent storage.
	Store(ctx context.Context, tx domain.Transaction) error

	// FindByAddress retrieves all stored transactions (both inbound and outbound).
	FindByAddress(ctx context.Context, address domain.Address) ([]domain.Transaction, error)
}
