package parser_state_test

import (
	"context"
	"errors"
	"testing"
	"trust_wallet_homework/internal/adapters/storage/memory/parser_state"

	"trust_wallet_homework/internal/core/domain"
	"trust_wallet_homework/internal/core/domain/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryParserStateRepo_GetSetCurrentBlock(t *testing.T) {
	repo := parser_state.NewInMemoryParserStateRepo()
	ctx := context.Background()

	_, err := repo.GetCurrentBlock(ctx)
	require.Error(t, err, "GetCurrentBlock() should return an error on initial call")
	assert.True(t, errors.Is(err, repository.ErrStateNotInitialized), "Error should be ErrStateNotInitialized")

	wantBlockNum1 := int64(100)
	block1, errBlock1 := domain.NewBlockNumber(wantBlockNum1)
	require.NoError(t, errBlock1, "Failed to create block number 100")
	err = repo.SetCurrentBlock(ctx, block1)
	require.NoError(t, err, "SetCurrentBlock() for block 100 failed")

	gotBlock1, err := repo.GetCurrentBlock(ctx)
	require.NoError(t, err, "GetCurrentBlock() after set 1 failed")
	assert.Equal(t, block1, gotBlock1, "GetCurrentBlock() after set 1 returned wrong block")

	wantBlockNum2 := int64(200)
	block2, errBlock2 := domain.NewBlockNumber(wantBlockNum2)
	require.NoError(t, errBlock2, "Failed to create block number 200")
	err = repo.SetCurrentBlock(ctx, block2)
	require.NoError(t, err, "SetCurrentBlock() for block 200 failed")

	gotBlock2, err := repo.GetCurrentBlock(ctx)
	require.NoError(t, err, "GetCurrentBlock() after set 2 failed")
	assert.Equal(t, block2, gotBlock2, "GetCurrentBlock() after set 2 returned wrong block")
}
