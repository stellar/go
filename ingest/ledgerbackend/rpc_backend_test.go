package ledgerbackend

import (
	"context"
	"fmt"
	"testing"
	"time"

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

func (m *MockRPCClient) GetHealth(ctx context.Context) (protocol.GetHealthResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(protocol.GetHealthResponse), args.Error(1)
}

func setupRPCTest(t *testing.T) (*RPCLedgerBackend, *MockRPCClient) {
	mockClient := new(MockRPCClient)
	backend, err := NewRPCLedgerBackend(mockClient, 0)
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
	mockHealth := protocol.GetHealthResponse{
		OldestLedger: 12345,
		LatestLedger: 12345 + 10,
	}
	mockClient.On("GetHealth", ctx).Return(mockHealth, nil).Once()

	preparedRange := Range{from: sequence, to: sequence + 10, bounded: true}
	rpcBackend.PrepareRange(ctx, preparedRange)

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

	mockSuccessResponse := protocol.GetLedgersResponse{
		Ledgers: []protocol.LedgerInfo{
			{
				Sequence:       sequence,
				LedgerMetadata: encodedLCM,
			},
		},
		LatestLedger: sequence + 10,
	}

	mockMissingLedgerResponse := protocol.GetLedgersResponse{
		Ledgers:      []protocol.LedgerInfo{},
		LatestLedger: sequence + 10,
	}

	expectedReq := protocol.GetLedgersRequest{
		StartLedger: sequence,
		Pagination: &protocol.LedgerPaginationOptions{
			Limit: uint(RPCBackendDefaultBufferSize),
		},
	}

	// Test ledger found response
	mockClient.On("GetLedgers", ctx, expectedReq).Return(mockSuccessResponse, nil).Once()
	actualLCM, err := rpcBackend.GetLedger(ctx, sequence)
	assert.NoError(t, err)
	assert.Equal(t, sequence, uint32(actualLCM.V0.LedgerHeader.Header.LedgerSeq))

	// Test requested ledger is after oldest range in rpc but, was missing from response
	notFoundSequnce := sequence + 1
	expectedReq.StartLedger = notFoundSequnce
	mockClient.On("GetLedgers", ctx, expectedReq).Return(mockMissingLedgerResponse, nil).Once()
	_, err = rpcBackend.GetLedger(ctx, notFoundSequnce)
	var missingErr *RPCLedgerMissingError
	assert.ErrorAs(t, err, &missingErr)
	assert.Equal(t, notFoundSequnce, missingErr.Sequence)

	// Test error response
	expectedErr := fmt.Errorf("rpc connection error")
	mockClient.On("GetLedgers", ctx, expectedReq).Return(protocol.GetLedgersResponse{}, expectedErr).Once()
	_, err = rpcBackend.GetLedger(ctx, sequence+1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), expectedErr.Error())
}

func TestRPCBackendImplementsInterface(t *testing.T) {
	var rpcBackend interface{} = (*RPCLedgerBackend)(nil)
	_, ok := rpcBackend.(LedgerBackend)
	assert.True(t, ok, "RPCLedgerBackend should implement LedgerBackend interface")
}

func TestNewRPCLedgerBackend(t *testing.T) {
	mockClient := new(MockRPCClient)

	t.Run("uses default buffer size when 0 provided", func(t *testing.T) {
		backend, err := NewRPCLedgerBackend(mockClient, 0)
		assert.NoError(t, err)
		assert.Equal(t, RPCBackendDefaultBufferSize, backend.bufferSize)
		assert.Equal(t, mockClient, backend.client)
	})

	t.Run("uses provided buffer size", func(t *testing.T) {
		customSize := uint32(20)
		backend, err := NewRPCLedgerBackend(mockClient, customSize)
		assert.NoError(t, err)
		assert.Equal(t, customSize, backend.bufferSize)
		assert.Equal(t, mockClient, backend.client)
	})
}

func TestGetLedgerBeyondLatest(t *testing.T) {
	rpcBackend, mockClient := setupRPCTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	requestedSequence := uint32(100)
	mockHealth := protocol.GetHealthResponse{
		OldestLedger: 100,
		LatestLedger: 200,
	}
	mockClient.On("GetHealth", ctx).Return(mockHealth, nil).Once()

	preparedRange := Range{from: requestedSequence, to: requestedSequence + 10, bounded: true}
	rpcBackend.PrepareRange(ctx, preparedRange)
	latestLedger := requestedSequence - 1 // Latest ledger is 1 behind requested

	rpcGetLedgersRequest := protocol.GetLedgersRequest{
		StartLedger: requestedSequence,
		Pagination: &protocol.LedgerPaginationOptions{
			Limit: uint(RPCBackendDefaultBufferSize),
		},
	}
	// Setup first response indicating ledger is beyond latest
	firstResponse := protocol.GetLedgersResponse{
		LatestLedger: latestLedger,
		Ledgers:      []protocol.LedgerInfo{}, // Empty ledgers array
	}
	mockClient.On("GetLedgers", ctx, rpcGetLedgersRequest).Return(firstResponse, nil).Once()

	// Setup second call to return the requested ledger
	lcm := xdr.LedgerCloseMeta{
		V: 0,
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					LedgerSeq: xdr.Uint32(requestedSequence),
				},
			},
		},
	}
	encodedLCM, err := xdr.MarshalBase64(lcm)
	assert.NoError(t, err)

	secondResponse := protocol.GetLedgersResponse{
		LatestLedger: requestedSequence,
		Ledgers: []protocol.LedgerInfo{
			{
				Sequence:       requestedSequence,
				LedgerMetadata: encodedLCM,
			},
		},
	}
	mockClient.On("GetLedgers", ctx, rpcGetLedgersRequest).Return(secondResponse, nil).Once()

	startTime := time.Now()
	actualLCM, err := rpcBackend.GetLedger(ctx, requestedSequence)
	duration := time.Since(startTime)

	assert.NoError(t, err)
	assert.Equal(t, requestedSequence, uint32(actualLCM.V0.LedgerHeader.Header.LedgerSeq))

	// Verify timing - GetLedger should have waited one interval and then refetched ledgers from rpc on second call
	assert.GreaterOrEqual(t, duration.Seconds(), float64(RPCBackendDefaultWaitIntervalSeconds))

}

func TestGetLedgerContextTimeout(t *testing.T) {
	rpcBackend, mockClient := setupRPCTest(t)
	sequence := uint32(100)
	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	mockHealth := protocol.GetHealthResponse{
		OldestLedger: 100,
		LatestLedger: 200,
	}
	mockClient.On("GetHealth", ctx).Return(mockHealth, nil).Once()

	preparedRange := Range{from: sequence, to: sequence + 10, bounded: true}
	rpcBackend.PrepareRange(ctx, preparedRange)

	// Setup mock to return "ledger beyond latest" response
	expectedReq := protocol.GetLedgersRequest{
		StartLedger: sequence,
		Pagination: &protocol.LedgerPaginationOptions{
			Limit: uint(RPCBackendDefaultBufferSize),
		},
	}
	mockResponse := protocol.GetLedgersResponse{
		LatestLedger: sequence - 2, // Make it return a beyond latest error
		Ledgers:      []protocol.LedgerInfo{},
	}
	mockClient.On("GetLedgers", ctx, expectedReq).Return(mockResponse, nil)

	// Call GetLedger and verify it returns context deadline exceeded
	_, err := rpcBackend.GetLedger(ctx, sequence)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestPrepareRange(t *testing.T) {
	t.Run("bounded range", func(t *testing.T) {
		backend, mockClient := setupRPCTest(t)
		ctx := context.Background()

		mockHealth := protocol.GetHealthResponse{
			OldestLedger: 100,
			LatestLedger: 200,
		}
		mockClient.On("GetHealth", ctx).Return(mockHealth, nil).Once()

		err := backend.PrepareRange(ctx, Range{from: 150, to: 180, bounded: true})
		assert.NoError(t, err)
		assert.Equal(t, uint32(150), backend.nextLedger)
	})

	t.Run("unbounded range", func(t *testing.T) {
		backend, mockClient := setupRPCTest(t)
		ctx := context.Background()

		mockHealth := protocol.GetHealthResponse{
			OldestLedger: 100,
			LatestLedger: 200,
		}
		mockClient.On("GetHealth", ctx).Return(mockHealth, nil).Once()

		err := backend.PrepareRange(ctx, Range{from: 150, bounded: false})
		assert.NoError(t, err)
		assert.Equal(t, uint32(150), backend.nextLedger)
	})

	t.Run("error when start is before oldest ledger", func(t *testing.T) {
		backend, mockClient := setupRPCTest(t)
		ctx := context.Background()

		mockHealth := protocol.GetHealthResponse{
			OldestLedger: 100,
			LatestLedger: 200,
		}
		mockClient.On("GetHealth", ctx).Return(mockHealth, nil).Once()

		err := backend.PrepareRange(ctx, Range{from: 50, to: 150, bounded: true})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "before oldest available ledger")
	})

	t.Run("error when end is beyond latest ledger", func(t *testing.T) {
		backend, mockClient := setupRPCTest(t)
		ctx := context.Background()

		mockHealth := protocol.GetHealthResponse{
			OldestLedger: 100,
			LatestLedger: 200,
		}
		mockClient.On("GetHealth", ctx).Return(mockHealth, nil).Once()

		err := backend.PrepareRange(ctx, Range{from: 150, to: 250, bounded: true})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "beyond latest available ledger")
	})

	t.Run("error when backend is closed", func(t *testing.T) {
		backend, _ := setupRPCTest(t)
		backend.closed = true

		err := backend.PrepareRange(context.Background(), Range{from: 1, to: 2})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "closed")
	})

	t.Run("error when already prepared", func(t *testing.T) {
		backend, mockClient := setupRPCTest(t)
		ctx := context.Background()

		mockHealth := protocol.GetHealthResponse{
			OldestLedger: 100,
			LatestLedger: 200,
		}
		mockClient.On("GetHealth", ctx).Return(mockHealth, nil).Once()

		// First prepare should succeed
		err := backend.PrepareRange(ctx, Range{from: 150, to: 180, bounded: true})
		assert.NoError(t, err)

		// Second prepare should fail
		err = backend.PrepareRange(ctx, Range{from: 160, to: 190, bounded: true})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already prepared")
	})

	t.Run("error when GetHealth fails", func(t *testing.T) {
		backend, mockClient := setupRPCTest(t)
		ctx := context.Background()

		expectedErr := fmt.Errorf("rpc connection error")
		mockClient.On("GetHealth", ctx).Return(protocol.GetHealthResponse{}, expectedErr).Once()

		err := backend.PrepareRange(ctx, Range{from: 150, to: 180, bounded: true})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
	})
}

func TestIsPrepared(t *testing.T) {
	t.Run("returns false when backend is not prepared", func(t *testing.T) {
		backend, _ := setupRPCTest(t)

		prepared, err := backend.IsPrepared(context.Background(), Range{from: 100, to: 200, bounded: true})
		assert.NoError(t, err)
		assert.False(t, prepared, "should return false when no range is prepared")
	})

	t.Run("returns true when ranges match exactly", func(t *testing.T) {
		backend, mockClient := setupRPCTest(t)
		ctx := context.Background()

		// First prepare a range
		mockHealth := protocol.GetHealthResponse{
			OldestLedger: 1,
			LatestLedger: 1000,
		}
		mockClient.On("GetHealth", ctx).Return(mockHealth, nil).Once()

		// Prepare initial range
		targetRange := Range{from: 100, to: 200, bounded: true}
		err := backend.PrepareRange(ctx, targetRange)
		assert.NoError(t, err)

		// Check same range
		prepared, err := backend.IsPrepared(ctx, targetRange)
		assert.NoError(t, err)
		assert.True(t, prepared, "should return true for exact range match")
	})

	t.Run("returns false when ranges differ", func(t *testing.T) {
		backend, mockClient := setupRPCTest(t)
		ctx := context.Background()

		// Prepare initial range
		mockHealth := protocol.GetHealthResponse{
			OldestLedger: 1,
			LatestLedger: 1000,
		}
		mockClient.On("GetHealth", ctx).Return(mockHealth, nil).Once()

		err := backend.PrepareRange(ctx, Range{from: 100, to: 200, bounded: true})
		assert.NoError(t, err)

		// Test different range variations
		testCases := []struct {
			name   string
			range_ Range
		}{
			{"different from", Range{from: 101, to: 200, bounded: true}},
			{"different to", Range{from: 100, to: 201, bounded: true}},
			{"different bounded", Range{from: 100, bounded: false}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				prepared, err := backend.IsPrepared(ctx, tc.range_)
				assert.NoError(t, err)
				assert.False(t, prepared, "should return false when ranges don't match exactly")
			})
		}
	})

	t.Run("returns error when backend is closed", func(t *testing.T) {
		backend, _ := setupRPCTest(t)
		backend.closed = true

		prepared, err := backend.IsPrepared(context.Background(), Range{from: 100, to: 200})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "closed")
		assert.False(t, prepared)
	})
}
