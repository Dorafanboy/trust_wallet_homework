// Package utils provides common utility functions.
package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// HexToInt64 converts a hex string (e.g., "0x1a") to int64.
func HexToInt64(hexStr string) (int64, error) {
	cleaned := strings.TrimPrefix(strings.ToLower(hexStr), "0x")
	if cleaned == "" {
		return 0, fmt.Errorf("empty hex string")
	}
	return strconv.ParseInt(cleaned, 16, 64)
}

// HexToUint64 converts a hex string (e.g., "0x1a") to uint64.
func HexToUint64(hexStr string) (uint64, error) {
	cleaned := strings.TrimPrefix(strings.ToLower(hexStr), "0x")
	if cleaned == "" {
		return 0, fmt.Errorf("empty hex string")
	}
	if cleaned == "0" {
		return 0, nil
	}
	return strconv.ParseUint(cleaned, 16, 64)
}
