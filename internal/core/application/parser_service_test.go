package application_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"trust_wallet_homework/internal/config"
	"trust_wallet_homework/internal/core/application"
	"trust_wallet_homework/internal/core/application/mocks/mock_client"
	"trust_wallet_homework/internal/core/application/mocks/mock_repository"
	"trust_wallet_homework/internal/core/domain"
	applogger "trust_wallet_homework/internal/logger"

	"github.com/stretchr/testify/assert"
)

func TestParserServiceImpl_GetCurrentBlock(t *testing.T) {
	service, mockStateRepo, _ := setupBasicService(t)

	ctx := context.Background()
	wantBlockNum := int64(12345)
	domainBlock, _ := domain.NewBlockNumber(wantBlockNum)

	mockStateRepo.On("GetCurrentBlock", ctx).Return(domainBlock, nil)

	got, err := service.GetCurrentBlock(ctx)
	assert.NoError(t, err)
	assert.Equal(t, wantBlockNum, got)

	mockStateRepo.AssertExpectations(t)
}

func TestParserServiceImpl_GetCurrentBlock_Error(t *testing.T) {
	service, mockStateRepo, _ := setupBasicService(t)

	ctx := context.Background()
	wantErr := errors.New("repo error")

	mockStateRepo.On("GetCurrentBlock", ctx).Return(domain.BlockNumber{}, wantErr)

	_, err := service.GetCurrentBlock(ctx)
	assert.Error(t, err)

	mockStateRepo.AssertExpectations(t)
}

func TestParserServiceImpl_Subscribe(t *testing.T) {
	service, _, mockAddrRepo := setupBasicService(t)

	ctx := context.Background()
	validAddrStr := "0x71c7656ec7ab88b098defb751b7401b5f6d8976f"
	domainAddr, _ := domain.NewAddress(validAddrStr)

	mockAddrRepo.On("Add", ctx, domainAddr).Return(nil)

	err := service.Subscribe(ctx, validAddrStr)
	assert.NoError(t, err)

	mockAddrRepo.AssertExpectations(t)
}

func TestParserServiceImpl_Subscribe_InvalidAddress(t *testing.T) {
	service, _, _ := setupBasicService(t)

	ctx := context.Background()
	invalidAddrStr := "0xinvalid"

	err := service.Subscribe(ctx, invalidAddrStr)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidAddressFormat), "Error should wrap domain.ErrInvalidAddressFormat")
}

func TestParserServiceImpl_Subscribe_RepoError(t *testing.T) {
	service, _, mockAddrRepo := setupBasicService(t)

	ctx := context.Background()
	validAddrStr := "0x71c7656ec7ab88b098defb751b7401b5f6d8976f"
	domainAddr, _ := domain.NewAddress(validAddrStr)
	wantErr := errors.New("repo error")

	mockAddrRepo.On("Add", ctx, domainAddr).Return(wantErr)

	err := service.Subscribe(ctx, validAddrStr)
	assert.Error(t, err)

	mockAddrRepo.AssertExpectations(t)
}

// setupBasicService is a helper for tests that primarily need the service, stateRepo and addrRepo.
func setupBasicService(t *testing.T) (
	*application.ParserServiceImpl,
	*mock_repository.ParserStateRepository,
	*mock_repository.MonitoredAddressRepository,
) {
	t.Helper()
	mockStateRepo := mock_repository.NewParserStateRepository(t)
	mockAddrRepo := mock_repository.NewMonitoredAddressRepository(t)
	mockTxRepo := mock_repository.NewTransactionRepository(t)
	mockEthClient := mock_client.NewEthereumClient(t)

	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	testAppLogger := applogger.NewSlogAdapter(discardLogger)

	cfg := config.ApplicationServiceConfig{
		PollingIntervalSeconds: 1,
	}

	service, err := application.NewParserService(
		mockStateRepo,
		mockAddrRepo,
		mockTxRepo,
		mockEthClient,
		testAppLogger,
		cfg,
	)
	if err != nil {
		t.Fatalf("Failed to create test service: %v", err)
	}

	return service, mockStateRepo, mockAddrRepo
}
