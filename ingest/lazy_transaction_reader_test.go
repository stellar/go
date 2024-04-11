package ingest

import (
	"io"
	"testing"

	"github.com/stellar/go/network"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	passphrase = network.TestNetworkPassphrase
	// Test prep:
	//   - two different envelopes which resolve to two different hashes
	//   - two basically-empty metas that contain the corresponding hashes
	//   - a ledger that has 5 txs with metas corresponding to these two envs
	//   - specifically, in the order [first, first, second, second, second]
	//
	// This tests both hash <--> envelope mapping and indexed iteration.
	txEnv1 = xdr.TransactionEnvelope{
		Type: xdr.EnvelopeTypeEnvelopeTypeTx,
		V1: &xdr.TransactionV1Envelope{
			Tx: xdr.Transaction{
				Ext:           xdr.TransactionExt{V: 0},
				SourceAccount: xdr.MustMuxedAddress("GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A"),
				Operations:    []xdr.Operation{},
				Fee:           123,
				SeqNum:        0,
			},
			Signatures: []xdr.DecoratedSignature{},
		},
	}
	txEnv2 = xdr.TransactionEnvelope{
		Type: xdr.EnvelopeTypeEnvelopeTypeTx,
		V1: &xdr.TransactionV1Envelope{
			Tx: xdr.Transaction{
				Ext:           xdr.TransactionExt{V: 0},
				SourceAccount: xdr.MustMuxedAddress("GCO26ZSBD63TKYX45H2C7D2WOFWOUSG5BMTNC3BG4QMXM3PAYI6WHKVZ"),
				Operations:    []xdr.Operation{},
				Fee:           456,
				SeqNum:        0,
			},
			Signatures: []xdr.DecoratedSignature{},
		},
	}
	txHash1, _ = network.HashTransactionInEnvelope(txEnv1, passphrase)
	txHash2, _ = network.HashTransactionInEnvelope(txEnv2, passphrase)
	txMeta1    = xdr.TransactionResultMeta{
		Result:            xdr.TransactionResultPair{TransactionHash: xdr.Hash(txHash1)},
		TxApplyProcessing: xdr.TransactionMeta{V: 3, V3: &xdr.TransactionMetaV3{}},
	}
	txMeta2 = xdr.TransactionResultMeta{
		Result:            xdr.TransactionResultPair{TransactionHash: xdr.Hash(txHash2)},
		TxApplyProcessing: xdr.TransactionMeta{V: 3, V3: &xdr.TransactionMetaV3{}},
	}
	// barebones LCM structure so that the tx reader works w/o nil derefs, 5 txs
	ledgerCloseMeta = xdr.LedgerCloseMeta{
		V: 1,
		V1: &xdr.LedgerCloseMetaV1{
			Ext: xdr.ExtensionPoint{V: 0},
			TxProcessing: []xdr.TransactionResultMeta{
				txMeta1,
				txMeta1,
				txMeta2,
				txMeta2,
				txMeta2,
			},
			TxSet: xdr.GeneralizedTransactionSet{
				V: 1,
				V1TxSet: &xdr.TransactionSetV1{
					Phases: []xdr.TransactionPhase{{
						V: 0,
						V0Components: &[]xdr.TxSetComponent{{
							TxsMaybeDiscountedFee: &xdr.TxSetComponentTxsMaybeDiscountedFee{
								Txs: []xdr.TransactionEnvelope{
									txEnv1,
									txEnv1,
									txEnv2,
									txEnv2,
									txEnv2,
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
)

func TestLazyTransactionReader(t *testing.T) {
	require.NotEqual(t,
		txHash1, txHash2,
		"precondition of different hashes violated: env1=%+v, env2=%+v",
		txEnv1, txEnv2)

	// simplest case: read from start

	fromZero, err := NewLazyTransactionReader(ledgerCloseMeta, passphrase, 0)
	require.NoError(t, err)
	for i := 0; i < 5; i++ {
		tx, ierr := fromZero.Read()
		require.NoError(t, ierr)
		assert.EqualValues(t, i+1, tx.Index, "iteration i=%d", i)

		thisHash, ierr := network.HashTransactionInEnvelope(tx.Envelope, passphrase)
		require.NoError(t, ierr)
		if tx.Index >= 3 {
			assert.Equal(t, txEnv2, tx.Envelope)
			assert.Equal(t, txHash2, thisHash)
		} else {
			assert.Equal(t, txEnv1, tx.Envelope)
			assert.Equal(t, txHash1, thisHash)
		}
	}
	_, err = fromZero.Read()
	require.ErrorIs(t, err, io.EOF)

	// start reading from the middle set of txs

	fromMiddle, err := NewLazyTransactionReader(ledgerCloseMeta, passphrase, 2)
	require.NoError(t, err)
	for i := 0; i < 5; i++ {
		tx, ierr := fromMiddle.Read()
		require.NoError(t, ierr)
		assert.EqualValues(t,
			/* txIndex is 1-based, iter is 0-based, start at 3rd tx, 5 total */
			1+(i+2)%5,
			tx.Index,
			"iteration i=%d", i)

		thisHash, ierr := network.HashTransactionInEnvelope(tx.Envelope, passphrase)
		require.NoError(t, ierr)
		if tx.Index >= 3 {
			assert.Equal(t, txEnv2, tx.Envelope)
			assert.Equal(t, txHash2, thisHash)
		} else {
			assert.Equal(t, txEnv1, tx.Envelope)
			assert.Equal(t, txHash1, thisHash)
		}
	}
	_, err = fromMiddle.Read()
	require.ErrorIs(t, err, io.EOF)

	// edge case: start from the last tx

	fromEnd, err := NewLazyTransactionReader(ledgerCloseMeta, passphrase, 4)
	require.NoError(t, err)
	for i := 0; i < 5; i++ {
		tx, ierr := fromEnd.Read()
		require.NoError(t, ierr)
		assert.EqualValues(t, 1+((i+4)%5), tx.Index, "iteration i=%d", i)

		thisHash, ierr := network.HashTransactionInEnvelope(tx.Envelope, passphrase)
		require.NoError(t, ierr)
		if tx.Index >= 3 {
			assert.Equal(t, txEnv2, tx.Envelope)
			assert.Equal(t, txHash2, thisHash)
		} else {
			assert.Equal(t, txEnv1, tx.Envelope)
			assert.Equal(t, txHash1, thisHash)
		}
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
	for _, idx := range []int{-1, 5, 6} {
		_, err = NewLazyTransactionReader(ledgerCloseMeta, passphrase, idx)
		require.Error(t, err, "no error when trying start=%d", idx)
	}
}
