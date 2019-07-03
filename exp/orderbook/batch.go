//lint:file-ignore U1000 this package is currently unused but it will be used in a future PR

package orderbook

import (
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
	// AddOffer will queue an operation to remove the given offer from the order book
	RemoveOffer(offerID xdr.Int64) BatchedUpdates
	// Apply will attempt to apply all the updates in the batch to the order book
	Apply() error
}

type orderBookOperation struct {
	operationType int
	offerID       xdr.Int64
	offer         *xdr.OfferEntry
}

type orderBookBatchedUpdates struct {
	operations []orderBookOperation
	orderbook  *OrderBookGraph
	committed  bool
}

// AddOffer will queue an operation to add the given offer to the order book
func (tx *orderBookBatchedUpdates) AddOffer(offer xdr.OfferEntry) BatchedUpdates {
	tx.operations = append(tx.operations, orderBookOperation{
		operationType: addOfferOperationType,
		offerID:       offer.OfferId,
		offer:         &offer,
	})

	return tx
}

// AddOffer will queue an operation to remove the given offer from the order book
func (tx *orderBookBatchedUpdates) RemoveOffer(offerID xdr.Int64) BatchedUpdates {
	tx.operations = append(tx.operations, orderBookOperation{
		operationType: removeOfferOperationType,
		offerID:       offerID,
	})

	return tx
}

// Apply will attempt to apply all the updates in the batch to the order book
func (tx *orderBookBatchedUpdates) Apply() error {
	tx.orderbook.lock.Lock()
	defer tx.orderbook.lock.Unlock()
	if tx.committed {
		return errBatchAlreadyApplied
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

	return nil
}
