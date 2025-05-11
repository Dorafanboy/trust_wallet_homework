package application

import (
	"trust_wallet_homework/internal/core/domain"
	"trust_wallet_homework/pkg/ethparser"
)

// mapDomainToAPITransaction converts an internal domain Transaction to the public API Transaction DTO.
func mapDomainToAPITransaction(domainTx domain.Transaction) ethparser.Transaction {
	return ethparser.Transaction{
		Hash:        domainTx.Hash.String(),
		From:        domainTx.From.String(),
		To:          domainTx.To.String(),
		Value:       domainTx.Value.String(),
		BlockNumber: domainTx.BlockNumber.Value(),
		Timestamp:   domainTx.Timestamp,
	}
}
