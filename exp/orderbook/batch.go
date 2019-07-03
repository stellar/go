package orderbook

import (
	"sync"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

const (
	_ = iota
	// the operationType enum values start at 1 because when constructing a
	// orderBookOperation struct, the operationType field should always be specified
	// explicity. if the operationType enum values started at 0 then it would be
	// possible to create a valid orderBookOperation struct without specifying
	// the operationType field
	addOfferOperationType    = iota
	removeOfferOperationType = iota
)

// BatchedUpdates is an interface for applying multiple
// operations on an order book
type BatchedUpdates interface {
	// AddOffer will queue an operation to add the given offer to the order book
	AddOffer(offer xdr.OfferEntry) BatchedUpdates
	// RemoveOffer will queue an operation to remove the given offer from the order book
	RemoveOffer(offerID xdr.Int64) BatchedUpdates
	// Apply will attempt to apply all the updates in the batch to the order book
	Apply()
}

type orderBookOperation struct {
	operationType int
	offerID       xdr.Int64
	offer         *xdr.OfferEntry
}

type orderBookBatchedUpdates struct {
	// mutex is protecting access to operations from multiple go routines
	mutex      sync.Mutex
	operations []orderBookOperation
	orderbook  *OrderBookGraph
	committed  bool
}

// AddOffer will queue an operation to add the given offer to the order book
func (tx *orderBookBatchedUpdates) AddOffer(offer xdr.OfferEntry) BatchedUpdates {
	tx.mutex.Lock()
	defer tx.mutex.Unlock()

	if tx.committed {
		panic(errBatchAlreadyApplied)
	}

	tx.operations = append(tx.operations, orderBookOperation{
		operationType: addOfferOperationType,
		offerID:       offer.OfferId,
		offer:         &offer,
	})

	return tx
}

// AddOffer will queue an operation to remove the given offer from the order book
func (tx *orderBookBatchedUpdates) RemoveOffer(offerID xdr.Int64) BatchedUpdates {
	tx.mutex.Lock()
	defer tx.mutex.Unlock()

	if tx.committed {
		panic(errBatchAlreadyApplied)
	}

	tx.operations = append(tx.operations, orderBookOperation{
		operationType: removeOfferOperationType,
		offerID:       offerID,
	})

	return tx
}

// Apply will attempt to apply all the updates in the batch to the order book
func (tx *orderBookBatchedUpdates) Apply() {
	tx.mutex.Lock()
	defer tx.mutex.Unlock()

	tx.orderbook.lock.Lock()
	defer tx.orderbook.lock.Unlock()

	if tx.committed {
		panic(errBatchAlreadyApplied)
	}
	tx.committed = true

	for _, operation := range tx.operations {
		switch operation.operationType {
		case addOfferOperationType:
			if err := tx.orderbook.add(*operation.offer); err != nil {
				panic(errors.Wrap(err, "could not apply update in batch"))
			}
		case removeOfferOperationType:
			if err := tx.orderbook.remove(operation.offerID); err != nil {
				panic(errors.Wrap(err, "could not apply update in batch"))
			}
		default:
			panic(errors.New("invalid operation type"))
		}
	}
}
