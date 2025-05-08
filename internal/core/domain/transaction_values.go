package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ErrInvalidTransactionHashFormat indicates invalid transaction hash format.
var ErrInvalidTransactionHashFormat = errors.New("invalid transaction hash format")

// Basic regex for Transaction Hash format validation (0x followed by 64 hex characters).
var ethTxHashRegex = regexp.MustCompile("^0x[0-9a-fA-F]{64}$")

// TransactionHash represents a validated transaction hash value object.
type TransactionHash struct {
	value string
}

// NewTransactionHash creates a new TransactionHash.
func NewTransactionHash(hash string) (TransactionHash, error) {
	cleanHash := strings.ToLower(strings.TrimSpace(hash))
	if !ethTxHashRegex.MatchString(cleanHash) {
		return TransactionHash{}, fmt.Errorf("%w: %s", ErrInvalidTransactionHashFormat, hash)
	}
	return TransactionHash{value: cleanHash}, nil
}

// String returns the string representation of the transaction hash.
func (th TransactionHash) String() string {
	return th.value
}

// IsZero checks if the TransactionHash is the zero value (empty).
func (th TransactionHash) IsZero() bool {
	return th.value == ""
}

// Equals checks if two TransactionHash objects are equal.
func (th TransactionHash) Equals(other TransactionHash) bool {
	return th.value == other.value
}

// WeiValue represents a transaction value, typically stored as a string
type WeiValue struct {
	value string
}

// NewWeiValue creates a new WeiValue object.
func NewWeiValue(value string) (WeiValue, error) {
	cleanedValue := strings.TrimSpace(value)
	return WeiValue{value: cleanedValue}, nil
}

// String returns the string representation of the wei value.
func (wv WeiValue) String() string {
	return wv.value
}
