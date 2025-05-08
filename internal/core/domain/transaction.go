package domain

// Transaction represents the core information about an Ethereum transaction.
type Transaction struct {
	Hash        TransactionHash
	From        Address
	To          Address
	Value       WeiValue
	BlockNumber BlockNumber
	Timestamp   uint64
}

// NewTransaction is a simple constructor for the Transaction entity.
func NewTransaction(
	hash TransactionHash,
	from Address,
	to Address,
	value WeiValue,
	blockNumber BlockNumber,
	timestamp uint64,
) Transaction {
	return Transaction{
		Hash:        hash,
		From:        from,
		To:          to,
		Value:       value,
		BlockNumber: blockNumber,
		Timestamp:   timestamp,
	}
}
