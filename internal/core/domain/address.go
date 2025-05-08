// Package domain defines the core domain models and business logic entities.
package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ErrInvalidAddressFormat indicates that the provided string is not a valid Ethereum address format.
var ErrInvalidAddressFormat = errors.New("invalid ethereum address format")

// Basic regex for Ethereum address format validation (0x followed by 40 hex characters).
var ethAddressRegex = regexp.MustCompile("^0x[0-9a-fA-F]{40}$")

// Address represents a validated Ethereum address value object.
type Address struct {
	value string
}

// NewAddress creates a new Address value object from a string.
func NewAddress(addr string) (Address, error) {
	cleanAddr := strings.ToLower(strings.TrimSpace(addr))

	if !ethAddressRegex.MatchString(cleanAddr) {
		return Address{}, fmt.Errorf("%w: %s", ErrInvalidAddressFormat, addr)
	}
	return Address{value: cleanAddr}, nil
}

// String returns the string representation of the address.
func (a Address) String() string {
	return a.value
}

// IsZero checks if the Address is the zero value (empty).
func (a Address) IsZero() bool {
	return a.value == ""
}

// Equals checks if two Address objects are equal.
func (a Address) Equals(other Address) bool {
	return a.value == other.value
}
