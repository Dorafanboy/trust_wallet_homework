// Package address provides an in-memory implementation of the MonitoredAddressRepository interface.
package address

import (
	"context"
	"sync"

	"trust_wallet_homework/internal/core/domain"
	"trust_wallet_homework/internal/core/domain/repository"
)

// InMemoryAddressRepo implements the MonitoredAddressRepository interface using an in-memory map.
type InMemoryAddressRepo struct {
	mu        sync.RWMutex
	addresses map[domain.Address]struct{}
}

// Compile-time check to ensure InMemoryAddressRepo implements repository.MonitoredAddressRepository
var _ repository.MonitoredAddressRepository = (*InMemoryAddressRepo)(nil)

// NewInMemoryAddressRepo creates a new in-memory address repository.
func NewInMemoryAddressRepo() *InMemoryAddressRepo {
	return &InMemoryAddressRepo{
		addresses: make(map[domain.Address]struct{}),
	}
}

// Add persists a new address to be monitored.
func (r *InMemoryAddressRepo) Add(_ context.Context, address domain.Address) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.addresses[address] = struct{}{}
	return nil
}

// Exists checks if a given address is already being monitored.
func (r *InMemoryAddressRepo) Exists(_ context.Context, address domain.Address) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.addresses[address]
	return exists, nil
}

// FindAll retrieves all addresses currently being monitored.
func (r *InMemoryAddressRepo) FindAll(_ context.Context) ([]domain.Address, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	addrList := make([]domain.Address, 0, len(r.addresses))
	for addr := range r.addresses {
		addrList = append(addrList, addr)
	}
	return addrList, nil
}
