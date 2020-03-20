package txsub

import (
	"context"
	"encoding/hex"

	"github.com/stellar/go/network"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type envelopeInfo struct {
	Hash          string
	Sequence      uint64
	SourceAddress string
}

func extractEnvelopeInfo(ctx context.Context, env string, passphrase string) (envelopeInfo, error) {
	var result envelopeInfo
	var tx xdr.TransactionEnvelope
	err := xdr.SafeUnmarshalBase64(env, &tx)
	if err != nil {
		return result, &MalformedTransactionError{env}
	}

	var hash [32]byte
	switch tx.Type {
	case xdr.EnvelopeTypeEnvelopeTypeTx:
		hash, err = network.HashTransaction(&tx.V1.Tx, passphrase)
	case xdr.EnvelopeTypeEnvelopeTypeTxV0:
		hash, err = network.HashTransactionV0(&tx.V0.Tx, passphrase)
	case xdr.EnvelopeTypeEnvelopeTypeTxFeeBump:
		hash, err = network.HashFeeBumpTransaction(&tx.FeeBump.Tx, passphrase)
	default:
		return result, errors.New("invalid transaction type")
	}
	if err != nil {
		return result, err
	}

	result.Hash = hex.EncodeToString(hash[:])
	result.Sequence = uint64(tx.SeqNum())
	sourceAccount := tx.SourceAccount()
	result.SourceAddress = sourceAccount.Address()
	return result, nil
}
