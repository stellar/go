package methods

import (
	"context"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/code"
	"github.com/creachadair/jrpc2/handler"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/network"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/xdr"
)

const (
	TransactionComplete = "complete"
	TransactionPending  = "pending"
)

type SendTransactionRequest struct {
	Transaction string `json:"transaction"`
}

type GetTransactionStatusRequest struct {
	Hash string `json:"hash"`
}

type TransactionStatusResponse struct {
	ID     string               `json:"id"`
	Status string               `json:"status"`
	Result *horizon.Transaction `json:"result"`
}

type SendTransactionResponse struct {
	ID string `json:"id"`
}

type transactionResult struct {
	timestamp time.Time
	pending   bool
	err       error
}

type horizonRequest struct {
	txHash         string
	transactionXDR string
}

type TransactionProxy struct {
	lock            sync.RWMutex
	results         map[string]transactionResult
	client          *horizonclient.Client
	passphrase      string
	queue           chan horizonRequest
	workers         int
	expiryFrequency time.Duration
	ttl             time.Duration
}

func NewTransactionProxy(
	client *horizonclient.Client,
	workers, queueSize int,
	networkPassphrase string,
	expiryFrequency, ttl time.Duration) *TransactionProxy {
	return &TransactionProxy{
		results:         map[string]transactionResult{},
		client:          client,
		passphrase:      networkPassphrase,
		queue:           make(chan horizonRequest, queueSize),
		workers:         workers,
		expiryFrequency: expiryFrequency,
		ttl:             ttl,
	}
}

func (p *TransactionProxy) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		go p.startWorker(ctx)
	}

	go p.startExpiryTicker(ctx)
}

func (p *TransactionProxy) SendTransaction(ctx context.Context, request SendTransactionRequest) (SendTransactionResponse, error) {
	var envelope xdr.TransactionEnvelope
	err := xdr.SafeUnmarshalBase64(request.Transaction, &envelope)
	if err != nil {
		return SendTransactionResponse{}, (&jrpc2.Error{
			Code:    code.InvalidParams,
			Message: "Invalid transaction XDR",
		}).WithData(err)
	}

	var hash [32]byte
	hash, err = network.HashTransactionInEnvelope(envelope, p.passphrase)
	if err != nil {
		return SendTransactionResponse{}, (&jrpc2.Error{
			Code:    code.InvalidParams,
			Message: "Transaction could not be hashed",
		}).WithData(err)
	}
	txHash := hex.EncodeToString(hash[:])

	p.lock.Lock()
	defer p.lock.Unlock()
	result, ok := p.results[txHash]
	// if pending or completed without any errors use
	// getTransactionStatus method with tx hash to obtain
	// response
	if result.pending || (ok && result.err == nil) {
		return SendTransactionResponse{
			ID: txHash,
		}, nil
	}

	p.results[txHash] = transactionResult{pending: true}
	select {
	case p.queue <- horizonRequest{txHash: txHash, transactionXDR: request.Transaction}:
		return SendTransactionResponse{
			ID: txHash,
		}, nil
	default:
		delete(p.results, txHash)
		return SendTransactionResponse{}, &jrpc2.Error{
			Code:    code.InternalError,
			Message: "Request queue is full",
		}
	}
}

func (p *TransactionProxy) setTxResult(txHash string, result transactionResult) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.results[txHash] = result
}

func (p *TransactionProxy) deletePendingEntry(txHash string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	delete(p.results, txHash)
}

func (p *TransactionProxy) startWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case request := <-p.queue:
			_, err := p.client.SubmitTransactionXDR(request.transactionXDR)
			if err != nil {
				result := transactionResult{timestamp: time.Now()}
				if herr, ok := err.(*horizonclient.Error); ok {
					result.err = (&jrpc2.Error{
						Code:    code.InvalidRequest,
						Message: herr.Problem.Title,
					}).WithData(herr.Problem.Extras)
				} else {
					result.err = err
				}
				p.setTxResult(request.txHash, result)
			} else {
				p.deletePendingEntry(request.txHash)
			}
		}
	}
}

func (p *TransactionProxy) GetTransactionStatus(ctx context.Context, request GetTransactionStatusRequest) (TransactionStatusResponse, error) {
	tx, err := p.client.TransactionDetail(request.Hash)
	if err != nil {
		if herr, ok := err.(*horizonclient.Error); ok {
			if herr.Problem.Status != http.StatusNotFound {
				return TransactionStatusResponse{}, (&jrpc2.Error{
					Code:    code.InvalidRequest,
					Message: herr.Problem.Title,
				}).WithData(herr.Problem.Extras)
			}
		} else {
			return TransactionStatusResponse{}, err
		}
	} else {
		return TransactionStatusResponse{
			ID:     request.Hash,
			Status: TransactionComplete,
			Result: &tx,
		}, nil
	}

	p.lock.RLock()
	defer p.lock.RUnlock()
	result, ok := p.results[request.Hash]
	if !ok {
		return TransactionStatusResponse{}, jrpc2.Error{
			Code:    code.InvalidRequest,
			Message: "Not Found",
		}
	}

	if result.pending {
		return TransactionStatusResponse{
			ID:     request.Hash,
			Status: TransactionPending,
			Result: nil,
		}, nil
	}

	return TransactionStatusResponse{}, result.err
}

func (p *TransactionProxy) startExpiryTicker(ctx context.Context) {
	ticker := time.NewTicker(p.expiryFrequency)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.deleteExpiredEntries(time.Now())
		case <-ctx.Done():
			return
		}
	}
}

func (p *TransactionProxy) deleteExpiredEntries(now time.Time) {
	p.lock.Lock()
	defer p.lock.Unlock()

	for key, val := range p.results {
		if !val.pending && now.Sub(val.timestamp) > p.ttl {
			delete(p.results, key)
		}
	}
}

// NewGetTransactionHandler returns a get transaction json rpc handler
func NewGetTransactionHandler(proxy *TransactionProxy) jrpc2.Handler {
	return handler.New(proxy.GetTransactionStatus)
}

// NewSendTransactionHandler returns a submit transaction json rpc handler
func NewSendTransactionHandler(proxy *TransactionProxy) jrpc2.Handler {
	return handler.New(proxy.SendTransaction)
}
