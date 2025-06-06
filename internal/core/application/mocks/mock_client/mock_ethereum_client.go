// Code generated by mockery v2.53.3. DO NOT EDIT.

package mock_client

import (
	context "context"
	domain "trust_wallet_homework/internal/core/domain"

	mock "github.com/stretchr/testify/mock"
)

// EthereumClient is an autogenerated mock type for the EthereumClient type
type EthereumClient struct {
	mock.Mock
}

// GetBlockWithTransactions provides a mock function with given fields: ctx, blockNumber
func (_m *EthereumClient) GetBlockWithTransactions(ctx context.Context, blockNumber domain.BlockNumber) (*domain.Block, error) {
	ret := _m.Called(ctx, blockNumber)

	if len(ret) == 0 {
		panic("no return value specified for GetBlockWithTransactions")
	}

	var r0 *domain.Block
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, domain.BlockNumber) (*domain.Block, error)); ok {
		return rf(ctx, blockNumber)
	}
	if rf, ok := ret.Get(0).(func(context.Context, domain.BlockNumber) *domain.Block); ok {
		r0 = rf(ctx, blockNumber)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.Block)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, domain.BlockNumber) error); ok {
		r1 = rf(ctx, blockNumber)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLatestBlockNumber provides a mock function with given fields: ctx
func (_m *EthereumClient) GetLatestBlockNumber(ctx context.Context) (domain.BlockNumber, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for GetLatestBlockNumber")
	}

	var r0 domain.BlockNumber
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (domain.BlockNumber, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) domain.BlockNumber); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(domain.BlockNumber)
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewEthereumClient creates a new instance of EthereumClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewEthereumClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *EthereumClient {
	mock := &EthereumClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
