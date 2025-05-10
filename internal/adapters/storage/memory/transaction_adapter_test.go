package memory_test

import (
	"context"
	"testing"

	"trust_wallet_homework/internal/adapters/storage/memory"
	"trust_wallet_homework/internal/core/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryTransactionRepo_Store_FindByAddress(t *testing.T) {
	repo := memory.NewInMemoryTransactionRepo()
	ctx := context.Background()

	addr1Str := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	addr2Str := "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	addr3Str := "0xcccccccccccccccccccccccccccccccccccccccc"
	addr1, err := domain.NewAddress(addr1Str)
	require.NoError(t, err)
	addr2, err := domain.NewAddress(addr2Str)
	require.NoError(t, err)
	addr3, err := domain.NewAddress(addr3Str)
	require.NoError(t, err)

	tx1Hash, err := domain.NewTransactionHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	require.NoError(t, err)
	tx2Hash, err := domain.NewTransactionHash("0x2222222222222222222222222222222222222222222222222222222222222222")
	require.NoError(t, err)
	tx3Hash, err := domain.NewTransactionHash("0x3333333333333333333333333333333333333333333333333333333333333333")
	require.NoError(t, err)

	val1, err := domain.NewWeiValue("0x1")
	require.NoError(t, err)
	val2, err := domain.NewWeiValue("2")
	require.NoError(t, err)
	val3, err := domain.NewWeiValue("0x3")
	require.NoError(t, err)

	block1, err := domain.NewBlockNumber(1)
	require.NoError(t, err)
	block2, err := domain.NewBlockNumber(2)
	require.NoError(t, err)

	tx1 := domain.NewTransaction(tx1Hash, addr1, addr2, val1, block1, 1000)
	tx2 := domain.NewTransaction(tx2Hash, addr2, addr3, val2, block1, 1001)
	tx3 := domain.NewTransaction(tx3Hash, addr1, addr3, val3, block2, 1002)

	txsAddr1Initial, err := repo.FindByAddress(ctx, addr1)
	require.NoError(t, err)
	assert.Empty(t, txsAddr1Initial)
	txsAddr2Initial, err := repo.FindByAddress(ctx, addr2)
	require.NoError(t, err)
	assert.Empty(t, txsAddr2Initial)
	txsAddr3Initial, err := repo.FindByAddress(ctx, addr3)
	require.NoError(t, err)
	assert.Empty(t, txsAddr3Initial)

	err = repo.Store(ctx, tx1)
	require.NoError(t, err)

	txsAddr1AfterTx1, err := repo.FindByAddress(ctx, addr1)
	require.NoError(t, err)
	assert.ElementsMatch(t, []domain.Transaction{tx1}, txsAddr1AfterTx1)

	txsAddr2AfterTx1, err := repo.FindByAddress(ctx, addr2)
	require.NoError(t, err)
	assert.ElementsMatch(t, []domain.Transaction{tx1}, txsAddr2AfterTx1)

	txsAddr3AfterTx1, err := repo.FindByAddress(ctx, addr3)
	require.NoError(t, err)
	assert.Empty(t, txsAddr3AfterTx1)

	err = repo.Store(ctx, tx2)
	require.NoError(t, err)

	txsAddr1AfterTx2, err := repo.FindByAddress(ctx, addr1)
	require.NoError(t, err)
	assert.ElementsMatch(t, []domain.Transaction{tx1}, txsAddr1AfterTx2)

	txsAddr2AfterTx2, err := repo.FindByAddress(ctx, addr2)
	require.NoError(t, err)
	assert.ElementsMatch(t, []domain.Transaction{tx1, tx2}, txsAddr2AfterTx2)

	txsAddr3AfterTx2, err := repo.FindByAddress(ctx, addr3)
	require.NoError(t, err)
	assert.ElementsMatch(t, []domain.Transaction{tx2}, txsAddr3AfterTx2)

	err = repo.Store(ctx, tx3)
	require.NoError(t, err)

	txsAddr1AfterTx3, err := repo.FindByAddress(ctx, addr1)
	require.NoError(t, err)
	assert.ElementsMatch(t, []domain.Transaction{tx1, tx3}, txsAddr1AfterTx3)

	txsAddr2AfterTx3, err := repo.FindByAddress(ctx, addr2)
	require.NoError(t, err)
	assert.ElementsMatch(t, []domain.Transaction{tx1, tx2}, txsAddr2AfterTx3)

	txsAddr3AfterTx3, err := repo.FindByAddress(ctx, addr3)
	require.NoError(t, err)
	assert.ElementsMatch(t, []domain.Transaction{tx2, tx3}, txsAddr3AfterTx3)
}
