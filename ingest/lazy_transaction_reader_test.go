package ingest

import (
	"io"
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var txMeta = xdr.TransactionResultMeta{
	TxApplyProcessing: xdr.TransactionMeta{
		V:  3,
		V3: &xdr.TransactionMetaV3{},
	},
}
var txEnv = xdr.TransactionEnvelope{
	Type: xdr.EnvelopeTypeEnvelopeTypeTx,
	V1: &xdr.TransactionV1Envelope{
		Tx: xdr.Transaction{},
	},
}

// barebones LCM structure so that the tx reader works w/o nil derefs, 5 txs
var ledgerCloseMeta = xdr.LedgerCloseMeta{
	V: 1,
	V1: &xdr.LedgerCloseMetaV1{
		Ext: xdr.ExtensionPoint{V: 0},
		TxProcessing: []xdr.TransactionResultMeta{
			txMeta,
			txMeta,
			txMeta,
			txMeta,
			txMeta,
		},
		TxSet: xdr.GeneralizedTransactionSet{
			V: 1,
			V1TxSet: &xdr.TransactionSetV1{
				Phases: []xdr.TransactionPhase{{
					V: 0,
					V0Components: &[]xdr.TxSetComponent{{
						TxsMaybeDiscountedFee: &xdr.TxSetComponentTxsMaybeDiscountedFee{
							Txs: []xdr.TransactionEnvelope{
								txEnv,
								txEnv,
								txEnv,
								txEnv,
								txEnv,
							},
						},
					}},
				}},
			},
		},
		LedgerHeader: xdr.LedgerHeaderHistoryEntry{
			Header: xdr.LedgerHeader{LedgerVersion: 20},
		},
	},
}

func TestLazyTransactionReader(t *testing.T) {
	require.True(t, true)

	// simplest case: read from start

	fromZero, err := NewLazyTransactionReader(ledgerCloseMeta, 0)
	require.NoError(t, err)
	for i := 0; i < 5; i++ {
		tx, ierr := fromZero.Read()
		require.NoError(t, ierr)
		assert.EqualValues(t, i+1, tx.Index, "iteration i=%d", i)
	}
	_, err = fromZero.Read()
	require.ErrorIs(t, err, io.EOF)

	// start reading from the middle set of txs

	fromMiddle, err := NewLazyTransactionReader(ledgerCloseMeta, 2)
	require.NoError(t, err)
	for i := 0; i < 5; i++ {
		tx, ierr := fromMiddle.Read()
		require.NoError(t, ierr)
		assert.EqualValues(t, 1+((i+2)%5), tx.Index, "iteration i=%d", i)
	}
	_, err = fromMiddle.Read()
	require.ErrorIs(t, err, io.EOF)

	// edge case: start from the last tx

	fromEnd, err := NewLazyTransactionReader(ledgerCloseMeta, 4)
	require.NoError(t, err)
	for i := 0; i < 5; i++ {
		tx, ierr := fromEnd.Read()
		require.NoError(t, ierr)
		assert.EqualValues(t, 1+((i+4)%5), tx.Index, "iteration i=%d", i)
	}
	_, err = fromEnd.Read()
	require.ErrorIs(t, err, io.EOF)

	// ensure that rewinds work after EOF

	fromEnd.Rewind()
	for i := 0; i < 5; i++ {
		tx, ierr := fromEnd.Read()
		require.NoError(t, ierr)
		assert.EqualValues(t, 1+((i+4)%5), tx.Index, "iteration i=%d", i)
	}
	_, err = fromEnd.Read()
	require.ErrorIs(t, err, io.EOF)

	// error case: too far or too close
	for i := range []int{-1, 5, 6} {
		_, err = NewLazyTransactionReader(ledgerCloseMeta, i)
		require.Error(t, err)
	}
}
