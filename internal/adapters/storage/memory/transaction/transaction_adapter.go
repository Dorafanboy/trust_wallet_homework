// Package transaction provides an in-memory implementation of the TransactionRepository interface.
package transaction

import (
	"context"
	"sync"

	"trust_wallet_homework/internal/core/domain"
	"trust_wallet_homework/internal/core/domain/repository"
)

// InMemoryTransactionRepo implements the TransactionRepository interface using in-memory storage.
type InMemoryTransactionRepo struct {
	mu           sync.RWMutex
	transactions map[string][]domain.Transaction
}

// Compile-time check to ensure InMemoryTransactionRepo implements repository.TransactionRepository
var _ repository.TransactionRepository = (*InMemoryTransactionRepo)(nil)

// NewInMemoryTransactionRepo creates a new in-memory transaction repository.
func NewInMemoryTransactionRepo() *InMemoryTransactionRepo {
	return &InMemoryTransactionRepo{
		transactions: make(map[string][]domain.Transaction),
	}
}

// Store saves a transaction to the persistent storage.
func (r *InMemoryTransactionRepo) Store(_ context.Context, tx domain.Transaction) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	fromAddr := tx.From.String()
	r.transactions[fromAddr] = append(r.transactions[fromAddr], tx)

	toAddr := tx.To.String()
	if toAddr != "" && !tx.To.IsZero() {
		if fromAddr != toAddr {
			r.transactions[toAddr] = append(r.transactions[toAddr], tx)
		}
	}
	return nil
}

// FindByAddress retrieves all stored transactions (both inbound and outbound)
func (r *InMemoryTransactionRepo) FindByAddress(
	_ context.Context,
	address domain.Address,
) ([]domain.Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	addrStr := address.String()
	txs, exists := r.transactions[addrStr]
	if !exists {
		return []domain.Transaction{}, nil
	}

	txCopy := make([]domain.Transaction, len(txs))
	copy(txCopy, txs)

	return txCopy, nil
}
