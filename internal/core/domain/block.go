package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrNegativeBlockNumber indicates that an attempt was made to create or use negative value block number.
	ErrNegativeBlockNumber = errors.New("block number cannot be negative")

	// ErrInvalidBlockHashFormat indicates that a provided string does not conform to the expected block hash.
	ErrInvalidBlockHashFormat = errors.New("invalid block hash format")
)

// Basic regex for Block Hash format validation (0x followed by 64 hex characters).
var ethBlockHashRegex = regexp.MustCompile("^0x[0-9a-fA-F]{64}$")

// BlockNumber represents a block number value object.
type BlockNumber struct {
	value int64
}

// NewBlockNumber creates a new BlockNumber.
func NewBlockNumber(number int64) (BlockNumber, error) {
	if number < 0 {
		return BlockNumber{}, fmt.Errorf("%w: %d", ErrNegativeBlockNumber, number)
	}
	return BlockNumber{value: number}, nil
}

// Value returns the int64 representation of the block number.
func (bn BlockNumber) Value() int64 {
	return bn.value
}

// BlockHash represents a validated block hash value object.
type BlockHash struct {
	value string
}

// NewBlockHash creates a new BlockHash.
func NewBlockHash(hash string) (BlockHash, error) {
	cleanHash := strings.ToLower(strings.TrimSpace(hash))
	if !ethBlockHashRegex.MatchString(cleanHash) {
		return BlockHash{}, fmt.Errorf("%w: %s", ErrInvalidBlockHashFormat, hash)
	}
	return BlockHash{value: cleanHash}, nil
}

// String returns the string representation of the block hash.
func (bh BlockHash) String() string {
	return bh.value
}

// IsZero checks if the BlockHash is the zero value (empty).
func (bh BlockHash) IsZero() bool {
	return bh.value == ""
}

// Equals checks if two BlockHash objects are equal.
func (bh BlockHash) Equals(other BlockHash) bool {
	return bh.value == other.value
}

// Block represents the core information about an Ethereum block.
type Block struct {
	Number       BlockNumber
	Hash         BlockHash
	Timestamp    uint64
	Transactions []Transaction
}

// NewBlock is a simple constructor for the Block entity.
func NewBlock(number BlockNumber, hash BlockHash, timestamp uint64, transactions []Transaction) Block {
	return Block{
		Number:       number,
		Hash:         hash,
		Timestamp:    timestamp,
		Transactions: transactions,
	}
}
