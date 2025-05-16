package ledgerbackend

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stellar/go/xdr"
	rpc "github.com/stellar/stellar-rpc/client"
	"github.com/stellar/stellar-rpc/protocol"
)

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
	client RPCClient
}

// Creates the RPCLedgerBackend with the given RPCClient.
func NewRPCLedgerBackend(client RPCClient) (*RPCLedgerBackend, error) {
	return &RPCLedgerBackend{
		client: client,
	}, nil
}

// Creates the RPCLedgerBackend with the given RPC URL and optional HTTP client.
func NewRPCLedgerBackendFromURL(rpcURL string, httpClient *http.Client) (*RPCLedgerBackend, error) {
	return NewRPCLedgerBackend(rpc.NewClient(rpcURL, httpClient))
}

func (b *RPCLedgerBackend) GetLatestLedgerSequence(ctx context.Context) (sequence uint32, err error) {
	ledger, err := b.client.GetLatestLedger(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest ledger sequence: %w", err)
	}
	return ledger.Sequence, nil
}

func (b *RPCLedgerBackend) GetLedger(ctx context.Context, sequence uint32) (xdr.LedgerCloseMeta, error) {
	req := protocol.GetLedgersRequest{
		StartLedger: sequence,
		Pagination: &protocol.LedgerPaginationOptions{
			Limit: 1,
		},
	}
	ledgers, err := b.client.GetLedgers(ctx, req)
	if err != nil {
		return xdr.LedgerCloseMeta{}, fmt.Errorf("failed to get ledger %d: %w", sequence, err)
	}

	if len(ledgers.Ledgers) == 0 {
		return xdr.LedgerCloseMeta{}, fmt.Errorf("ledger %d not found", sequence)
	}

	var lcm xdr.LedgerCloseMeta
	if err := xdr.SafeUnmarshalBase64(ledgers.Ledgers[0].LedgerMetadata, &lcm); err != nil {
		return xdr.LedgerCloseMeta{}, fmt.Errorf("failed to unmarshal ledger close meta: %w", err)
	}

	return lcm, nil
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
