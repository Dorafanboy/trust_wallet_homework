// Package parser_state provides an in-memory implementation of the ParserStateRepository interface.
package parser_state

import (
	"context"
	"sync"

	"trust_wallet_homework/internal/core/domain"
	"trust_wallet_homework/internal/core/domain/repository"
)

// InMemoryParserStateRepo is an in-memory implementation of ParserStateRepository.
type InMemoryParserStateRepo struct {
	mu               sync.RWMutex
	lastScannedBlock *domain.BlockNumber
}

// Compile-time check to ensure InMemoryParserStateRepo implements repository.ParserStateRepository
var _ repository.ParserStateRepository = (*InMemoryParserStateRepo)(nil)

// NewInMemoryParserStateRepo creates a new InMemoryParserStateRepo.
func NewInMemoryParserStateRepo() *InMemoryParserStateRepo {
	return &InMemoryParserStateRepo{}
}

// GetCurrentBlock retrieves the last scanned block number.
func (r *InMemoryParserStateRepo) GetCurrentBlock(_ context.Context) (domain.BlockNumber, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.lastScannedBlock == nil {
		return domain.BlockNumber{}, repository.ErrStateNotInitialized
	}
	return *r.lastScannedBlock, nil
}

// SetCurrentBlock stores the last scanned block number.
func (r *InMemoryParserStateRepo) SetCurrentBlock(_ context.Context, blockNumber domain.BlockNumber) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	bnCopy := blockNumber
	r.lastScannedBlock = &bnCopy
	return nil
}
