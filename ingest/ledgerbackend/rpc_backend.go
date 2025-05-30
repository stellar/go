package ledgerbackend

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/stellar/go/xdr"
	rpc "github.com/stellar/stellar-rpc/client"
	"github.com/stellar/stellar-rpc/protocol"
)

const RPCBackendDefaultBufferSize uint32 = 10
const RPCBackendDefaultWaitIntervalSeconds uint32 = 2

type RPCLedgerMissingError struct {
	Sequence uint32
}

func (e *RPCLedgerMissingError) Error() string {
	return fmt.Sprintf("ledger %d was not present on rpc", e.Sequence)
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
	GetHealth(ctx context.Context) (protocol.GetHealthResponse, error)
}

// RPCLedgerBackend does not support stateful range preparations.
// The rpc backend is composed of ephermeral sliding window of ledgers and therefore
// connot prepare a range of ledgers which remains consistent over time.
//
// Callers should focus on using RPCLedgerBackend.GetLedger for the ledger range needed
// and check the returned error for presence of a ledger.
//
// Instances of RPCLedgerBackend are thread-safe and can be used concurrently across goroutines.
type RPCLedgerBackend struct {
	client        RPCClient
	buffer        map[uint32]xdr.LedgerCloseMeta
	bufferSize    uint32
	preparedRange *Range
	nextLedger    uint32
	closed        bool
	backendLock   sync.RWMutex
}

// NewRPCLedgerBackend creates a new RPCLedgerBackend instance that fetches ledger data
// from a Stellar RPC server.
//
// Parameters:
//   - client: The RPC client implementation used to communicate with the server
//   - bufferSize: Size of the ledger retrieval buffer (in number of ledgers).
//     If 0, defaults to 10.
//
// Returns:
//   - *RPCLedgerBackend: A new backend instance ready for use
//   - error: If initialization fails
//
// The returned backend must be prepared with PrepareRange before GetLedger can be called.
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

// NewRPCLedgerBackendFromURL creates a new RPCLedgerBackend instance using the provided RPC URL.
//
// Parameters:
//   - rpcURL: URL of the Stellar RPC server - "https://rpc_host:8000")
//   - httpClient: Optional custom HTTP client. If nil, a default client will be used
//   - bufferSize: Size of the ledger buffer (in number of ledgers). If 0, defaults to 10
//
// Returns:
//   - *RPCLedgerBackend: A new backend instance ready for use
//   - error: If initialization fails
//
// The returned backend must be prepared with PrepareRange before GetLedger can be called.
func NewRPCLedgerBackendFromURL(rpcURL string, httpClient *http.Client, bufferSize uint32) (*RPCLedgerBackend, error) {
	return NewRPCLedgerBackend(rpc.NewClient(rpcURL, httpClient), bufferSize)
}

// GetLatestLedgerSequence queries the RPC server for the latest ledger sequence.
func (b *RPCLedgerBackend) GetLatestLedgerSequence(ctx context.Context) (sequence uint32, err error) {
	b.backendLock.RLock()
	defer b.backendLock.RUnlock()

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
//   - nil                      if ledger is found
//   - RPCLedgerNotFoundError   if sequence falls within the RPC's retention window,
//     but the ledger for sequence is not in RPC's retained history
//   - context.DeadlineExceeded if context times out
//   - context.Canceled         if context is cancelled
//   - Error                    if sequence falls outside of RPC's current retention window
//     or due to any other error in general
func (b *RPCLedgerBackend) GetLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error) {
	b.backendLock.RLock()
	defer b.backendLock.RUnlock()

	if err := b.checkClosed(); err != nil {
		return xdr.LedgerCloseMeta{}, err
	}

	if b.preparedRange == nil {
		return xdr.LedgerCloseMeta{}, fmt.Errorf("RPCLedgerBackend must be prepared before calling GetLedger")
	}

	if sequence < b.preparedRange.from || (b.preparedRange.bounded && sequence > b.preparedRange.to) {
		return xdr.LedgerCloseMeta{}, fmt.Errorf("requested ledger %d is outside prepared range [%d, %d]",
			sequence, b.preparedRange.from, b.preparedRange.to)
	}

	if sequence != b.nextLedger {
		return xdr.LedgerCloseMeta{}, fmt.Errorf("requested ledger %d is not the expected ledger %d", sequence, b.nextLedger)
	}

	for {
		lcm, err := b.getBufferedLedger(ctx, sequence)
		if err == nil {
			b.nextLedger = sequence + 1
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

// PrepareRange validates that the requested ledger range is within the RPC server's
// current history window by checking the RPC health endpoint.
// It cannot gaurantee ledgers within this range will be available when requested later by GetLedger.
// See Also: GetLedger for more details on how the RPCLedgerBackend handles ledger availability.
func (b *RPCLedgerBackend) PrepareRange(ctx context.Context, ledgerRange Range) error {
	b.backendLock.RLock()
	defer b.backendLock.RUnlock()

	if err := b.checkClosed(); err != nil {
		return err
	}

	if b.preparedRange != nil {
		return fmt.Errorf("RPCLedgerBackend is already prepared with range [%d, %d]", b.preparedRange.from, b.preparedRange.to)
	}
	health, err := b.client.GetHealth(ctx)
	if err != nil {
		return fmt.Errorf("failed to get RPC health info: %w", err)
	}

	if ledgerRange.from < health.OldestLedger {
		return fmt.Errorf("requested range start ledger %d is before oldest available ledger %d",
			ledgerRange.from, health.OldestLedger)
	}

	// Check bounded range end is not beyond latest ledger
	if ledgerRange.bounded && ledgerRange.to > health.LatestLedger {
		return fmt.Errorf("requested range end ledger %d is beyond latest available ledger %d",
			ledgerRange.to, health.LatestLedger)
	}
	b.nextLedger = ledgerRange.from
	b.preparedRange = &ledgerRange
	return nil
}

func (b *RPCLedgerBackend) IsPrepared(ctx context.Context, ledgerRange Range) (bool, error) {
	b.backendLock.RLock()
	defer b.backendLock.RUnlock()

	if err := b.checkClosed(); err != nil {
		return false, err
	}

	if b.preparedRange == nil {
		return false, nil
	}

	rangesMatch := b.preparedRange.from == ledgerRange.from &&
		b.preparedRange.bounded == ledgerRange.bounded &&
		(!b.preparedRange.bounded || b.preparedRange.to == ledgerRange.to)

	return rangesMatch, nil
}

func (b *RPCLedgerBackend) Close() error {
	b.backendLock.RLock()
	defer b.backendLock.RUnlock()

	b.closed = true
	return nil
}

func (b *RPCLedgerBackend) checkClosed() error {
	if b.closed {
		return fmt.Errorf("RPCLedgerBackend is closed")
	}
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

	return xdr.LedgerCloseMeta{}, &RPCLedgerMissingError{Sequence: sequence}
}
