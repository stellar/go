package network

import (
	"testing"

	"github.com/xdbfoundation/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashTransaction(t *testing.T) {
	var txe xdr.TransactionEnvelope
	err := xdr.SafeUnmarshalBase64("AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAACgAAAAAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAAAO5rKAAAAAAAAAAABVvwF9wAAAEAKZ7IPj/46PuWU6ZOtyMosctNAkXRNX9WCAI5RnfRk+AyxDLoDZP/9l3NvsxQtWj9juQOuoBlFLnWu8intgxQA", &txe)

	require.NoError(t, err)

	expected := [32]byte{
		0x39, 0x6f, 0xeb, 0xd, 0xc6, 0x7c, 0x18, 0x16, 0x4b, 0xb8, 0x9c, 0xe7, 0xb9, 0x44, 0x10, 0x52, 0x41, 0xdf, 0x1e, 0xfb, 0xb6, 0x71, 0xb3, 0xed, 0xc8, 0x81, 0xa0, 0x8b, 0x17, 0x96, 0x85, 0x88
	}
	actual, err := HashTransactionV0(txe.V0.Tx, TestNetworkPassphrase)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	actual, err = HashTransactionInEnvelope(txe, TestNetworkPassphrase)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	_, err = HashTransactionV0(txe.V0.Tx, "")
	assert.Contains(t, err.Error(), "empty network passphrase")
	_, err = HashTransactionInEnvelope(txe, "")
	assert.Contains(t, err.Error(), "empty network passphrase")

	tx := xdr.Transaction{
		SourceAccount: txe.SourceAccount(),
		Fee:           xdr.Uint32(txe.Fee()),
		Memo:          txe.Memo(),
		Operations:    txe.Operations(),
		SeqNum:        xdr.SequenceNumber(txe.SeqNum()),
		TimeBounds:    txe.TimeBounds(),
	}
	actual, err = HashTransaction(tx, TestNetworkPassphrase)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	txe.Type = xdr.EnvelopeTypeEnvelopeTypeTx
	txe.V0 = nil
	txe.V1 = &xdr.TransactionV1Envelope{
		Tx: tx,
	}
	actual, err = HashTransactionInEnvelope(txe, TestNetworkPassphrase)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	// sadpath: empty passphrase
	_, err = HashTransaction(tx, "")
	assert.Contains(t, err.Error(), "empty network passphrase")
	_, err = HashTransactionInEnvelope(txe, "")
	assert.Contains(t, err.Error(), "empty network passphrase")

	sourceAID := xdr.MustAddress("GCLOMB72ODBFUGK4E2BK7VMR3RNZ5WSTMEOGNA2YUVHFR3WMH2XBAB6H")
	feeBumpTx := xdr.FeeBumpTransaction{
		Fee:       123456,
		FeeSource: sourceAID.ToMuxedAccount(),
		InnerTx: xdr.FeeBumpTransactionInnerTx{
			Type: xdr.EnvelopeTypeEnvelopeTypeTx,
			V1: &xdr.TransactionV1Envelope{
				Tx:         tx,
				Signatures: []xdr.DecoratedSignature{},
			},
		},
	}

	expected = [32]uint8{
		0xcf, 0x5d, 0xae, 0x9b, 0xf9, 0xbe, 0x7c, 0x6d, 0xe3, 0xbb, 0xc, 0x18, 0xc8, 0x45, 0x5d, 0x75, 0x21, 0xf0, 0xc9, 0xdd, 0x6e, 0x97, 0x2e, 0x27, 0xe5, 0x6a, 0x7d, 0x52, 0x93, 0x5f, 0x5f, 0xa4
	}
	actual, err = HashFeeBumpTransaction(feeBumpTx, TestNetworkPassphrase)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	txe.Type = xdr.EnvelopeTypeEnvelopeTypeTxFeeBump
	txe.V1 = nil
	txe.FeeBump = &xdr.FeeBumpTransactionEnvelope{
		Tx: feeBumpTx,
	}
	actual, err = HashTransactionInEnvelope(txe, TestNetworkPassphrase)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	_, err = HashFeeBumpTransaction(feeBumpTx, "")
	assert.Contains(t, err.Error(), "empty network passphrase")
	_, err = HashTransactionInEnvelope(txe, "")
	assert.Contains(t, err.Error(), "empty network passphrase")
}
