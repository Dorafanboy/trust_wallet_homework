package address_test

import (
	"context"
	"testing"
	"trust_wallet_homework/internal/adapters/storage/memory/address"

	"trust_wallet_homework/internal/core/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryAddressRepo_AddExistsFindAll(t *testing.T) {
	repo := address.NewInMemoryAddressRepo()
	ctx := context.Background()

	addr1Str := "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	addr2Str := "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	addr1, err1 := domain.NewAddress(addr1Str)
	require.NoError(t, err1)
	addr2, err2 := domain.NewAddress(addr2Str)
	require.NoError(t, err2)

	initialAddrs, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Empty(t, initialAddrs)

	exists1, err := repo.Exists(ctx, addr1)
	require.NoError(t, err)
	assert.False(t, exists1)
	exists2, err := repo.Exists(ctx, addr2)
	require.NoError(t, err)
	assert.False(t, exists2)

	err = repo.Add(ctx, addr1)
	require.NoError(t, err)

	exists1, err = repo.Exists(ctx, addr1)
	require.NoError(t, err)
	assert.True(t, exists1)

	addrsAfter1, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, addrsAfter1, 1)
	assert.Contains(t, addrsAfter1, addr1)

	err = repo.Add(ctx, addr2)
	require.NoError(t, err)

	err = repo.Add(ctx, addr1)
	require.NoError(t, err)

	exists1, err = repo.Exists(ctx, addr1)
	require.NoError(t, err)
	assert.True(t, exists1)
	exists2, err = repo.Exists(ctx, addr2)
	require.NoError(t, err)
	assert.True(t, exists2)

	addrsAfter2, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, addrsAfter2, 2)
	assert.ElementsMatch(t, []domain.Address{addr1, addr2}, addrsAfter2)
}
