package queue

import (
	"math/big"

	"github.com/stellar/go/support/errors"
)

type AssetCode string

const (
	AssetCodeETH AssetCode = "ETH"
)

var (
	ten      = big.NewInt(10)
	eighteen = big.NewInt(18)
	// weiInEth = 10^18
	weiInEth = new(big.Rat).SetInt(new(big.Int).Exp(ten, eighteen, nil))
)

type Transaction struct {
	TransactionID string
	AssetCode     AssetCode
	// Amount in the smallest unit of currency.
	// For 1 satoshi = 0.00000001 BTC this should be equal `1`
	// For 1 Wei = 0.000000000000000001 ETH this should be equal `1`
	Amount           string
	StellarPublicKey string
}

func (t Transaction) AmountToEth(prec int) (string, error) {
	if t.AssetCode != AssetCodeETH {
		return "", errors.New("Asset code not ETH")
	}

	valueWei := new(big.Int)
	_, ok := valueWei.SetString(t.Amount, 10)
	if !ok {
		return "", errors.Errorf("%s is not a valid integer", t.Amount)
	}
	valueEth := new(big.Rat).Quo(new(big.Rat).SetInt(valueWei), weiInEth)
	return valueEth.FloatString(prec), nil
}

// Queue implements transactions queue.
// The queue must not allow duplicates (including history) or implement deduplication
// interval so it should not allow duplicate entries for 5 minutes since the first
// entry with the same ID was added.
// This is a critical requirement! Otherwise ETH/BTC may be sent twice to Stellar account.
// If you don't know what to do, use default AWS SQS FIFO queue or DB queue.
type Queue interface {
	// Add inserts the element to this queue.
	Add(tx Transaction) error
	// Pool receives and removes the head of this queue. Returns nil if no elements found.
	Pool() (*Transaction, error)
}

type SQSFiFo struct{}
