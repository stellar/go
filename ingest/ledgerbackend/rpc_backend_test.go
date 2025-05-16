package ledgerbackend

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/xdr"
	"github.com/stellar/stellar-rpc/protocol"
)

type MockRPCClient struct {
	mock.Mock
}

func (m *MockRPCClient) GetLatestLedger(ctx context.Context) (protocol.GetLatestLedgerResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(protocol.GetLatestLedgerResponse), args.Error(1)
}

func (m *MockRPCClient) GetLedgers(ctx context.Context, req protocol.GetLedgersRequest) (protocol.GetLedgersResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(protocol.GetLedgersResponse), args.Error(1)
}

func setupRPCTest(t *testing.T) (*RPCBackend, *MockRPCClient) {
	mockClient := new(MockRPCClient)
	backend, err := NewRPCBackend(mockClient)
	assert.NoError(t, err)
	return backend, mockClient
}

func TestRPCGetLatestLedgerSequence(t *testing.T) {

	rpcBackend, mockClient := setupRPCTest(t)
	ctx := context.Background()
	expectedSequence := uint32(12345)
	mockResponse := protocol.GetLatestLedgerResponse{
		Sequence: expectedSequence,
		Hash:     "hasge",
	}
	mockClient.On("GetLatestLedger", ctx).Return(mockResponse, nil)
	sequence, err := rpcBackend.GetLatestLedgerSequence(ctx)
	assert.NoError(t, err)
	assert.Equal(t, expectedSequence, sequence)

}
func TestRPCGetLedger(t *testing.T) {
	rpcBackend, mockClient := setupRPCTest(t)
	ctx := context.Background()
	sequence := uint32(12345)

	lcm := xdr.LedgerCloseMeta{
		V: 0,
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					LedgerSeq: xdr.Uint32(sequence),
				},
			},
		},
	}

	encodedLCM, err := xdr.MarshalBase64(lcm)
	assert.NoError(t, err)

	mockResponse := protocol.GetLedgersResponse{
		Ledgers: []protocol.LedgerInfo{
			{
				Sequence:       sequence,
				LedgerMetadata: encodedLCM,
			},
		},
	}

	expectedReq := protocol.GetLedgersRequest{
		StartLedger: sequence,
		Pagination: &protocol.LedgerPaginationOptions{
			Limit: 1,
		},
	}

	// Test successful case
	mockClient.On("GetLedgers", ctx, expectedReq).Return(mockResponse, nil).Once()
	actualLCM, err := rpcBackend.GetLedger(ctx, sequence)
	assert.NoError(t, err)
	assert.Equal(t, sequence, uint32(actualLCM.V0.LedgerHeader.Header.LedgerSeq))

	// Test empty response
	mockClient.On("GetLedgers", ctx, expectedReq).Return(protocol.GetLedgersResponse{}, nil).Once()
	_, err = rpcBackend.GetLedger(ctx, sequence)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test error response
	expectedErr := fmt.Errorf("rpc connection error")
	mockClient.On("GetLedgers", ctx, expectedReq).Return(protocol.GetLedgersResponse{}, expectedErr).Once()
	_, err = rpcBackend.GetLedger(ctx, sequence)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), expectedErr.Error())
}
