package resourceadapter

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/guregu/null"
	"github.com/lib/pq"
	"github.com/stellar/go/xdr"

	. "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/test"
	stellarTime "github.com/stellar/go/support/time"
	"github.com/stretchr/testify/assert"
)

// TestPopulateTransaction_Successful tests transaction object population.
func TestPopulateTransaction_Successful(t *testing.T) {
	ctx, _ := test.ContextWithLogBuffer()

	var (
		dest Transaction
		row  history.Transaction
	)

	dest = Transaction{}
	row = history.Transaction{
		TransactionWithoutLedger: history.TransactionWithoutLedger{
			Successful: true,
		},
	}

	assert.NoError(t, PopulateTransaction(ctx, row.TransactionHash, &dest, row))
	assert.True(t, dest.Successful)

	dest = Transaction{}
	row = history.Transaction{
		TransactionWithoutLedger: history.TransactionWithoutLedger{
			Successful: false,
		},
	}

	assert.NoError(t, PopulateTransaction(ctx, row.TransactionHash, &dest, row))
	assert.False(t, dest.Successful)
}

func TestPopulateTransaction_HashMemo(t *testing.T) {
	ctx, _ := test.ContextWithLogBuffer()
	dest := Transaction{}
	row := history.Transaction{
		TransactionWithoutLedger: history.TransactionWithoutLedger{
			MemoType: "hash",
			Memo:     null.StringFrom("abcdef"),
		},
	}
	assert.NoError(t, PopulateTransaction(ctx, row.TransactionHash, &dest, row))
	assert.Equal(t, "hash", dest.MemoType)
	assert.Equal(t, "abcdef", dest.Memo)
	assert.Equal(t, "", dest.MemoBytes)
}

func TestPopulateTransaction_TextMemo(t *testing.T) {
	ctx, _ := test.ContextWithLogBuffer()
	rawMemo := []byte{0, 0, 1, 1, 0, 0, 3, 3}
	rawMemoString := string(rawMemo)

	sourceAID := xdr.MustAddress("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")
	feeSourceAID := xdr.MustAddress("GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU")
	for _, envelope := range []xdr.TransactionEnvelope{
		{
			Type: xdr.EnvelopeTypeEnvelopeTypeTxV0,
			V0: &xdr.TransactionV0Envelope{
				Tx: xdr.TransactionV0{
					Memo: xdr.Memo{
						Type: xdr.MemoTypeMemoText,
						Text: &rawMemoString,
					},
				},
			},
		},
		{
			Type: xdr.EnvelopeTypeEnvelopeTypeTx,
			V1: &xdr.TransactionV1Envelope{
				Tx: xdr.Transaction{
					SourceAccount: sourceAID.ToMuxedAccount(),
					Memo: xdr.Memo{
						Type: xdr.MemoTypeMemoText,
						Text: &rawMemoString,
					},
				},
			},
		},
		{
			Type: xdr.EnvelopeTypeEnvelopeTypeTxFeeBump,
			FeeBump: &xdr.FeeBumpTransactionEnvelope{
				Tx: xdr.FeeBumpTransaction{
					InnerTx: xdr.FeeBumpTransactionInnerTx{
						Type: xdr.EnvelopeTypeEnvelopeTypeTx,
						V1: &xdr.TransactionV1Envelope{
							Tx: xdr.Transaction{
								SourceAccount: sourceAID.ToMuxedAccount(),
								Memo: xdr.Memo{
									Type: xdr.MemoTypeMemoText,
									Text: &rawMemoString,
								},
							},
						},
					},
					FeeSource: feeSourceAID.ToMuxedAccount(),
				},
			},
		},
	} {
		envelopeXDR, err := xdr.MarshalBase64(envelope)
		assert.NoError(t, err)
		row := history.Transaction{
			TransactionWithoutLedger: history.TransactionWithoutLedger{
				MemoType:   "text",
				TxEnvelope: envelopeXDR,
				Memo:       null.StringFrom("sample"),
			},
		}

		var dest Transaction
		assert.NoError(t, PopulateTransaction(ctx, row.TransactionHash, &dest, row))

		assert.Equal(t, "text", dest.MemoType)
		assert.Equal(t, "sample", dest.Memo)
		assert.Equal(t, base64.StdEncoding.EncodeToString(rawMemo), dest.MemoBytes)
	}
}

// TestPopulateTransaction_Fee tests transaction object population.
func TestPopulateTransaction_Fee(t *testing.T) {
	ctx, _ := test.ContextWithLogBuffer()

	var (
		dest Transaction
		row  history.Transaction
	)

	dest = Transaction{}
	row = history.Transaction{
		TransactionWithoutLedger: history.TransactionWithoutLedger{
			MaxFee:     10000,
			FeeCharged: 100,
		},
	}

	assert.NoError(t, PopulateTransaction(ctx, row.TransactionHash, &dest, row))
	assert.Equal(t, int64(100), dest.FeeCharged)
	assert.Equal(t, int64(10000), dest.MaxFee)
}

// TestPopulateTransaction_Preconditions tests transaction object population.
func TestPopulateTransaction_Preconditions(t *testing.T) {
	ctx, _ := test.ContextWithLogBuffer()

	var (
		dest Transaction
		row  history.Transaction
	)

	validAfter := time.Now().Add(-1 * time.Hour)
	validBefore := time.Now().Add(1 * time.Hour)
	minLedger := uint32(40071006 - 1024)
	maxLedger := uint32(40071006 + 1024)
	minAccountSequence := int64(10)
	minSequenceAge := 30 * time.Second * 1000
	minSequenceLedgerGap := uint32(5)

	dest = Transaction{}
	row = history.Transaction{
		TransactionWithoutLedger: history.TransactionWithoutLedger{
			TimeBounds: history.TimeBounds{
				Lower: null.IntFrom(stellarTime.MillisFromTime(validAfter).ToInt64() / 1000),
				Upper: null.IntFrom(stellarTime.MillisFromTime(validBefore).ToInt64() / 1000),
			},
			LedgerBounds: history.LedgerBounds{
				MinLedger: null.IntFrom(int64(minLedger)),
				MaxLedger: null.IntFrom(int64(maxLedger)),
			},
			MinAccountSequence:   null.IntFrom(minAccountSequence),
			MinSequenceAge:       null.IntFrom(int64(minSequenceAge)),
			MinSequenceLedgerGap: null.IntFrom(int64(minSequenceLedgerGap)),
			ExtraSigners:         pq.StringArray{"D34DB33F", "8BADF00D"},
		},
	}

	assert.NoError(t, PopulateTransaction(ctx, row.TransactionHash, &dest, row))
	p := dest.Preconditions
	assert.Equal(t, validAfter.Format(time.RFC3339), dest.ValidAfter)
	assert.Equal(t, validBefore.Format(time.RFC3339), dest.ValidBefore)
	assert.Equal(t, validAfter.Format(time.RFC3339), p.Timebounds.MinTime)
	assert.Equal(t, validBefore.Format(time.RFC3339), p.Timebounds.MaxTime)
	assert.Equal(t, &minLedger, p.Ledgerbounds.MinLedger)
	assert.Equal(t, &maxLedger, p.Ledgerbounds.MaxLedger)
	assert.Equal(t, fmt.Sprint(minAccountSequence), p.MinAccountSequence)
	assert.Equal(t, fmt.Sprint(int64(minSequenceAge)), p.MinSequenceAge)
	assert.Equal(t, minSequenceLedgerGap, p.MinSequenceLedgerGap)
	assert.Equal(t, []string{"D34DB33F", "8BADF00D"}, p.ExtraSigners)
}

func TestPopulateTransaction_PreconditionsV2(t *testing.T) {
	ctx, _ := test.ContextWithLogBuffer()

	sourceAID := xdr.MustAddress("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")
	feeSourceAID := xdr.MustAddress("GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU")
	timebounds := &xdr.TimeBounds{
		MinTime: 5,
		MaxTime: 10,
	}
	for _, envelope := range []xdr.TransactionEnvelope{
		{
			Type: xdr.EnvelopeTypeEnvelopeTypeTxV0,
			V0: &xdr.TransactionV0Envelope{
				Tx: xdr.TransactionV0{
					TimeBounds: timebounds,
				},
			},
		},
		{
			Type: xdr.EnvelopeTypeEnvelopeTypeTx,
			V1: &xdr.TransactionV1Envelope{
				Tx: xdr.Transaction{
					SourceAccount: sourceAID.ToMuxedAccount(),
					Cond: xdr.Preconditions{
						Type:       xdr.PreconditionTypePrecondTime,
						TimeBounds: timebounds,
					},
				},
			},
		},
		{
			Type: xdr.EnvelopeTypeEnvelopeTypeTx,
			V1: &xdr.TransactionV1Envelope{
				Tx: xdr.Transaction{
					SourceAccount: sourceAID.ToMuxedAccount(),
					Cond: xdr.Preconditions{
						Type: xdr.PreconditionTypePrecondV2,
						V2: &xdr.PreconditionsV2{
							TimeBounds: timebounds,
						},
					},
				},
			},
		},
		{
			Type: xdr.EnvelopeTypeEnvelopeTypeTxFeeBump,
			FeeBump: &xdr.FeeBumpTransactionEnvelope{
				Tx: xdr.FeeBumpTransaction{
					InnerTx: xdr.FeeBumpTransactionInnerTx{
						Type: xdr.EnvelopeTypeEnvelopeTypeTx,
						V1: &xdr.TransactionV1Envelope{
							Tx: xdr.Transaction{
								SourceAccount: sourceAID.ToMuxedAccount(),
								Cond: xdr.Preconditions{
									Type: xdr.PreconditionTypePrecondV2,
									V2: &xdr.PreconditionsV2{
										TimeBounds: timebounds,
									},
								},
							},
						},
					},
					FeeSource: feeSourceAID.ToMuxedAccount(),
				},
			},
		},
	} {
		envelopeXDR, err := xdr.MarshalBase64(envelope)
		assert.NoError(t, err)
		envelopeTimebounds := envelope.Preconditions().TimeBounds
		row := history.Transaction{
			TransactionWithoutLedger: history.TransactionWithoutLedger{
				TimeBounds: history.TimeBounds{
					Lower: null.IntFrom(int64(envelopeTimebounds.MinTime)),
					Upper: null.IntFrom(int64(envelopeTimebounds.MaxTime)),
				},
				TxEnvelope: envelopeXDR,
			},
		}

		var dest Transaction
		assert.NoError(t, PopulateTransaction(ctx, row.TransactionHash, &dest, row))

		gotTimebounds := dest.Preconditions.Timebounds
		assert.Equal(t, "1970-01-01T00:00:05Z", gotTimebounds.MinTime)
		assert.Equal(t, "1970-01-01T00:00:10Z", gotTimebounds.MaxTime)
	}
}

func TestFeeBumpTransaction(t *testing.T) {
	ctx, _ := test.ContextWithLogBuffer()
	dest := Transaction{}
	row := history.Transaction{
		TransactionWithoutLedger: history.TransactionWithoutLedger{
			MaxFee:               123,
			FeeCharged:           100,
			TransactionHash:      "cebb875a00ff6e1383aef0fd251a76f22c1f9ab2a2dffcb077855736ade2659a",
			FeeAccount:           null.StringFrom("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ"),
			FeeAccountMuxed:      null.StringFrom("MA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUAAAAAAAAAAAACJUQ"),
			Account:              "GAQAA5L65LSYH7CQ3VTJ7F3HHLGCL3DSLAR2Y47263D56MNNGHSQSTVY",
			AccountMuxed:         null.StringFrom("MAQAA5L65LSYH7CQ3VTJ7F3HHLGCL3DSLAR2Y47263D56MNNGHSQSAAAAAAAAAAE2LP26"),
			NewMaxFee:            null.IntFrom(10000),
			InnerTransactionHash: null.StringFrom("2374e99349b9ef7dba9a5db3339b78fda8f34777b1af33ba468ad5c0df946d4d"),
			Signatures:           []string{"a", "b", "c"},
			InnerSignatures:      []string{"d", "e", "f"},
		},
	}

	assert.NoError(t, PopulateTransaction(ctx, row.TransactionHash, &dest, row))
	assert.Equal(t, row.TransactionHash, dest.Hash)
	assert.Equal(t, row.TransactionHash, dest.ID)
	assert.Equal(t, row.FeeAccount.String, dest.FeeAccount)
	assert.Equal(t, row.FeeAccountMuxed.String, dest.FeeAccountMuxed)
	assert.Equal(t, uint64(0), dest.FeeAccountMuxedID)
	assert.Equal(t, row.Account, dest.Account)
	assert.Equal(t, row.AccountMuxed.String, dest.AccountMuxed)
	assert.Equal(t, uint64(1234), dest.AccountMuxedID)
	assert.Equal(t, row.FeeCharged, dest.FeeCharged)
	assert.Equal(t, row.NewMaxFee.Int64, dest.MaxFee)
	assert.Equal(t, []string{"a", "b", "c"}, dest.Signatures)
	assert.Equal(t, row.InnerTransactionHash.String, dest.InnerTransaction.Hash)
	assert.Equal(t, row.MaxFee, dest.InnerTransaction.MaxFee)
	assert.Equal(t, []string{"d", "e", "f"}, dest.InnerTransaction.Signatures)
	assert.Equal(t, row.TransactionHash, dest.FeeBumpTransaction.Hash)
	assert.Equal(t, []string{"a", "b", "c"}, dest.FeeBumpTransaction.Signatures)
	assert.Equal(t, "/transactions/"+row.TransactionHash, dest.Links.Transaction.Href)

	assert.NoError(t, PopulateTransaction(ctx, row.InnerTransactionHash.String, &dest, row))
	assert.Equal(t, row.InnerTransactionHash.String, dest.Hash)
	assert.Equal(t, row.InnerTransactionHash.String, dest.ID)
	assert.Equal(t, row.FeeAccount.String, dest.FeeAccount)
	assert.Equal(t, row.FeeAccountMuxed.String, dest.FeeAccountMuxed)
	assert.Equal(t, uint64(0), dest.FeeAccountMuxedID)
	assert.Equal(t, row.Account, dest.Account)
	assert.Equal(t, row.AccountMuxed.String, dest.AccountMuxed)
	assert.Equal(t, uint64(1234), dest.AccountMuxedID)
	assert.Equal(t, row.FeeCharged, dest.FeeCharged)
	assert.Equal(t, row.NewMaxFee.Int64, dest.MaxFee)
	assert.Equal(t, []string{"d", "e", "f"}, dest.Signatures)
	assert.Equal(t, row.InnerTransactionHash.String, dest.InnerTransaction.Hash)
	assert.Equal(t, row.MaxFee, dest.InnerTransaction.MaxFee)
	assert.Equal(t, []string{"d", "e", "f"}, dest.InnerTransaction.Signatures)
	assert.Equal(t, row.TransactionHash, dest.FeeBumpTransaction.Hash)
	assert.Equal(t, []string{"a", "b", "c"}, dest.FeeBumpTransaction.Signatures)
	assert.Equal(t, "/transactions/"+row.InnerTransactionHash.String, dest.Links.Transaction.Href)
}
