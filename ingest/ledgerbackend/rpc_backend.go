package ledgerbackend

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/stellar/go/xdr"
	rpc "github.com/stellar/stellar-rpc/client"
	"github.com/stellar/stellar-rpc/protocol"
)

const RPCBackendDefaultBufferSize uint32 = 10
const RPCBackendDefaultWaitIntervalSeconds uint32 = 2

type RPCLedgerNotFoundError struct {
	Sequence uint32
}

func (e *RPCLedgerNotFoundError) Error() string {
	return fmt.Sprintf("ledger %d not found", e.Sequence)
}

type RPCLedgerBeyondLatestError struct {
	Sequence     uint32
	LatestLedger uint32
}

func (e *RPCLedgerBeyondLatestError) Error() string {
	return fmt.Sprintf("ledger %d is beyond the RPC latest ledger is %d", e.Sequence, e.LatestLedger)
}

// The minimum required RPC client interface required for usage by RPCLedgerBackend.
type RPCClient interface {
	GetLatestLedger(ctx context.Context) (protocol.GetLatestLedgerResponse, error)
	GetLedgers(ctx context.Context, req protocol.GetLedgersRequest) (protocol.GetLedgersResponse, error)
}

// RPCLedgerBackend does not support stateful range preparations.
// The rpc backend is composed of ephermeral sliding window of ledgers and therefore
// connot determinsitcally prepare a range of ledgers.
// Callers should focus on using RPCLedgerBackend.GetLedger for the ledger range needed
// and check the returned error for presence of a ledger.
type RPCLedgerBackend struct {
	client     RPCClient
	buffer     map[uint32]xdr.LedgerCloseMeta
	bufferSize uint32
}

// Creates the RPCLedgerBackend with the given RPCClient.
func NewRPCLedgerBackend(client RPCClient, bufferSize uint32) (*RPCLedgerBackend, error) {
	if bufferSize == 0 {
		bufferSize = RPCBackendDefaultBufferSize
	}
	backend := &RPCLedgerBackend{
		client:     client,
		bufferSize: bufferSize,
	}
	backend.initBuffer()
	return backend, nil
}

// Creates the RPCLedgerBackend with the given RPC URL and optional HTTP client.
func NewRPCLedgerBackendFromURL(rpcURL string, httpClient *http.Client, bufferSize uint32) (*RPCLedgerBackend, error) {
	return NewRPCLedgerBackend(rpc.NewClient(rpcURL, httpClient), bufferSize)
}

// GetLatestLedgerSequence queries the RPC server for the latest ledger sequence.
func (b *RPCLedgerBackend) GetLatestLedgerSequence(ctx context.Context) (sequence uint32, err error) {
	ledger, err := b.client.GetLatestLedger(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest ledger sequence: %w", err)
	}
	return ledger.Sequence, nil
}

// GetLedger queries the RPC server for a specific ledger sequence and returns the meta data.
// If the requested ledger is not immediately available but is beyond the latest ledger in the RPC server,
// it will block and retry until either:
//   - The ledger becomes available
//   - The context is cancelled or times out
//   - An error occurs
//
// The caller can control the maximum wait time by setting a timeout or deadline on the provided context.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - sequence: The ledger sequence number to retrieve meta data for
//
// Returns:
//   - xdr.LedgerCloseMeta: The ledger meta data if found
//   - error: One of:
//   - nil if ledger is found
//   - RPCLedgerNotFoundError if ledger is not in RPC's retention window
//   - context.DeadlineExceeded if context times out
//   - context.Canceled if context is cancelled
//   - Other errors that may be due to RPC usage or network issues
func (b *RPCLedgerBackend) GetLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error) {
	for {
		lcm, err := b.getBufferedLedger(ctx, sequence)
		if err == nil {
			return lcm, nil
		}

		_, isBeyondErr := err.(*RPCLedgerBeyondLatestError)
		if !isBeyondErr {
			return xdr.LedgerCloseMeta{}, err
		}

		select {
		case <-ctx.Done():
			return xdr.LedgerCloseMeta{}, ctx.Err()
		case <-time.After(time.Duration(RPCBackendDefaultWaitIntervalSeconds) * time.Second):
			continue
		}
	}
}

// RPCLedgerBackend does not perform stateful range preparation.
// This has no effect on the backend and is a no-op.
func (b *RPCLedgerBackend) PrepareRange(ctx context.Context, ledgerRange Range) error {
	return nil
}

// RPCLedgerBackend does not perform stateful range preparation.
// This has no effect on the backend and is a no-op.
func (b *RPCLedgerBackend) IsPrepared(ctx context.Context, ledgerRange Range) (bool, error) {
	return false, nil
}

func (b *RPCLedgerBackend) Close() error {
	return nil
}

func (b *RPCLedgerBackend) initBuffer() {
	b.buffer = make(map[uint32]xdr.LedgerCloseMeta)
}

func (b *RPCLedgerBackend) getBufferedLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error) {
	// Check if ledger is in buffer
	if lcm, exists := b.buffer[sequence]; exists {
		return lcm, nil
	}

	// Ledger not in buffer, fetch a small batch from RPC starting from the requested sequence
	req := protocol.GetLedgersRequest{
		StartLedger: sequence,
		Pagination: &protocol.LedgerPaginationOptions{
			Limit: uint(b.bufferSize),
		},
	}

	ledgers, err := b.client.GetLedgers(ctx, req)
	if err != nil {
		return xdr.LedgerCloseMeta{}, fmt.Errorf("failed to get ledgers starting from %d: %w", sequence, err)
	}

	b.initBuffer()

	// Check if requested ledger is beyond the RPC retention window
	if sequence > ledgers.LatestLedger {
		return xdr.LedgerCloseMeta{}, &RPCLedgerBeyondLatestError{
			Sequence:     sequence,
			LatestLedger: ledgers.LatestLedger,
		}
	}

	// Populate buffer with new ledgers
	for _, ledger := range ledgers.Ledgers {
		var lcm xdr.LedgerCloseMeta
		if err := xdr.SafeUnmarshalBase64(ledger.LedgerMetadata, &lcm); err != nil {
			return xdr.LedgerCloseMeta{}, fmt.Errorf("failed to unmarshal ledger %d: %w", ledger.Sequence, err)
		}
		b.buffer[ledger.Sequence] = lcm
	}

	// Check if requested ledger is in new buffer
	if lcm, exists := b.buffer[sequence]; exists {
		return lcm, nil
	}

	return xdr.LedgerCloseMeta{}, &RPCLedgerNotFoundError{Sequence: sequence}
}
