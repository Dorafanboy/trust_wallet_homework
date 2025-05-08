// Package repository defines interfaces for data storage and retrieval operations.
//
//go:generate mockgen -source=$GOFILE -destination=../../mocks/mock_$GOPACKAGE/mock_$GOFILE -package=mock_$GOPACKAGE
package repository

import (
	"context"

	"trust_wallet_homework/internal/core/domain"
)

// MonitoredAddressRepository defines the interface for managing the set of addresses
type MonitoredAddressRepository interface {
	// Add persists a new address to be monitored.
	Add(ctx context.Context, address domain.Address) error

	// Exists checks if a given address is already being monitored.
	Exists(ctx context.Context, address domain.Address) (bool, error)

	// FindAll retrieves all addresses currently being monitored.
	FindAll(ctx context.Context) ([]domain.Address, error)
}
