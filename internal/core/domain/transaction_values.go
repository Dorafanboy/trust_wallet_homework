package domain

import (
	"errors"
	"fmt"
	"math/big"
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

// ErrInvalidWeiValueFormat indicates that the provided string is not a valid Wei value format.
var ErrInvalidWeiValueFormat = errors.New("invalid wei value format")

// WeiValue represents a transaction value, typically stored as a string
type WeiValue struct {
	value *big.Int
}

// NewWeiValue creates a new WeiValue object from a string.
func NewWeiValue(s string) (WeiValue, error) {
	trimmedStr := strings.TrimSpace(s)
	if trimmedStr == "" {
		return WeiValue{}, fmt.Errorf("%w: input string is empty", ErrInvalidWeiValueFormat)
	}

	val := new(big.Int)
	var ok bool

	if strings.HasPrefix(trimmedStr, "0x") || strings.HasPrefix(trimmedStr, "0X") {
		if len(trimmedStr) == 2 {
			return WeiValue{}, fmt.Errorf("%w: hex string is too short '%s'", ErrInvalidWeiValueFormat, trimmedStr)
		}
		_, ok = val.SetString(trimmedStr[2:], 16)
	} else {
		_, ok = val.SetString(trimmedStr, 10)
	}

	if !ok {
		return WeiValue{}, fmt.Errorf("%w: failed to parse '%s'", ErrInvalidWeiValueFormat, trimmedStr)
	}

	return WeiValue{value: val}, nil
}

// String returns the string representation of the wei value in hex format ("0x...").
func (wv WeiValue) String() string {
	if wv.value == nil {
		return "0x0"
	}
	return "0x" + wv.value.Text(16)
}

// BigInt returns a copy of the internal *big.Int value.
func (wv WeiValue) BigInt() *big.Int {
	if wv.value == nil {
		return big.NewInt(0)
	}
	valCopy := new(big.Int)
	valCopy.Set(wv.value)
	return valCopy
}

// IsZero checks if the WeiValue represents zero.
func (wv WeiValue) IsZero() bool {
	if wv.value == nil {
		return true
	}
	return wv.value.Sign() == 0
}

// Equals checks if two WeiValue objects are equal.
func (wv WeiValue) Equals(other WeiValue) bool {
	if wv.value == nil && other.value == nil {
		return true
	}
	if wv.value == nil || other.value == nil {
		return false
	}
	return wv.value.Cmp(other.value) == 0
}
