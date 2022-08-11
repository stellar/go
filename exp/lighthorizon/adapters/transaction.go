package adapters

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/stellar/go/exp/lighthorizon/archive"
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/network"
	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/xdr"
	"golang.org/x/exp/constraints"
)

// PopulateTransaction converts between ingested XDR and RESTful JSON. In
// Horizon Classic, the data goes from Captive Core -> DB -> JSON. In our case,
// there's no DB intermediary, so we need to directly translate.
func PopulateTransaction(
	baseUrl *url.URL,
	tx *common.Transaction,
	encoder *xdr.EncodingBuffer,
) (dest protocol.Transaction, err error) {
	txHash, err := tx.TransactionHash()
	if err != nil {
		return
	}

	dest.ID = txHash
	dest.Successful = tx.Result.Successful()
	dest.Hash = txHash
	dest.Ledger = int32(tx.LedgerHeader.LedgerSeq)
	dest.LedgerCloseTime = time.Unix(int64(tx.LedgerHeader.ScpValue.CloseTime), 0).UTC()

	source := tx.SourceAccount()
	dest.Account = source.ToAccountId().Address()
	if _, ok := source.GetMed25519(); ok {
		dest.AccountMuxed, err = source.GetAddress()
		if err != nil {
			return
		}
		dest.AccountMuxedID, err = source.GetId()
		if err != nil {
			return
		}
	}
	dest.AccountSequence = tx.Envelope.SeqNum()

	envelopeBase64, err := encoder.MarshalBase64(tx.Envelope)
	if err != nil {
		return
	}
	resultBase64, err := encoder.MarshalBase64(&tx.Result.Result)
	if err != nil {
		return
	}
	metaBase64, err := encoder.MarshalBase64(tx.UnsafeMeta)
	if err != nil {
		return
	}
	feeMetaBase64, err := encoder.MarshalBase64(tx.FeeChanges)
	if err != nil {
		return
	}

	dest.OperationCount = int32(len(tx.Envelope.Operations()))
	dest.EnvelopeXdr = envelopeBase64
	dest.ResultXdr = resultBase64
	dest.ResultMetaXdr = metaBase64
	dest.FeeMetaXdr = feeMetaBase64
	dest.MemoType = memoType(*tx.LedgerTransaction)
	if m, ok := memo(*tx.LedgerTransaction); ok {
		dest.Memo = m
		if dest.MemoType == "text" {
			var mb string
			if mb, err = memoBytes(envelopeBase64); err != nil {
				return
			} else {
				dest.MemoBytes = mb
			}
		}
	}

	dest.Signatures = signatures(tx.Envelope.Signatures())

	// If we never use this, we'll remove it later. This just defends us against
	// nil dereferences.
	dest.Preconditions = &protocol.TransactionPreconditions{}

	if tb := tx.Envelope.Preconditions().TimeBounds; tb != nil {
		dest.Preconditions.TimeBounds = &protocol.TransactionPreconditionsTimebounds{
			MaxTime: formatTime(tb.MaxTime),
			MinTime: formatTime(tb.MinTime),
		}
	}

	if lb := tx.Envelope.LedgerBounds(); lb != nil {
		dest.Preconditions.LedgerBounds = &protocol.TransactionPreconditionsLedgerbounds{
			MinLedger: uint32(lb.MinLedger),
			MaxLedger: uint32(lb.MaxLedger),
		}
	}

	if minSeq := tx.Envelope.MinSeqNum(); minSeq != nil {
		dest.Preconditions.MinAccountSequence = fmt.Sprint(*minSeq)
	}

	if minSeqAge := tx.Envelope.MinSeqAge(); minSeqAge != nil && *minSeqAge > 0 {
		dest.Preconditions.MinAccountSequenceAge = formatTime(*minSeqAge)
	}

	if minSeqGap := tx.Envelope.MinSeqLedgerGap(); minSeqGap != nil {
		dest.Preconditions.MinAccountSequenceLedgerGap = uint32(*minSeqGap)
	}

	if signers := tx.Envelope.ExtraSigners(); len(signers) > 0 {
		dest.Preconditions.ExtraSigners = formatSigners(signers)
	}

	if tx.Envelope.IsFeeBump() {
		innerTx, ok := tx.Envelope.FeeBump.Tx.InnerTx.GetV1()
		if !ok {
			panic("Failed to parse inner transaction from fee-bump tx.")
		}

		var rawInnerHash [32]byte
		rawInnerHash, err = network.HashTransaction(innerTx.Tx, tx.NetworkPassphrase)
		if err != nil {
			return
		}
		innerHash := hex.EncodeToString(rawInnerHash[:])

		feeAccountMuxed := tx.Envelope.FeeBumpAccount()
		dest.FeeAccount = feeAccountMuxed.ToAccountId().Address()
		if _, ok := feeAccountMuxed.GetMed25519(); ok {
			dest.FeeAccountMuxed, err = feeAccountMuxed.GetAddress()
			if err != nil {
				return
			}
			dest.FeeAccountMuxedID, err = feeAccountMuxed.GetId()
			if err != nil {
				return
			}
		}

		dest.MaxFee = tx.Envelope.FeeBumpFee()
		dest.FeeBumpTransaction = &protocol.FeeBumpTransaction{
			Hash:       txHash,
			Signatures: signatures(tx.Envelope.FeeBumpSignatures()),
		}
		dest.InnerTransaction = &protocol.InnerTransaction{
			Hash:       innerHash,
			MaxFee:     int64(innerTx.Tx.Fee),
			Signatures: signatures(tx.Envelope.Signatures()),
		}
		// TODO: Figure out what this means? Maybe @tamirms knows.
		// if transactionHash != row.TransactionHash {
		// 	dest.Signatures = dest.InnerTransaction.Signatures
		// }
	} else {
		dest.FeeAccount = dest.Account
		dest.FeeAccountMuxed = dest.AccountMuxed
		dest.FeeAccountMuxedID = dest.AccountMuxedID
		dest.MaxFee = int64(tx.Envelope.Fee())
	}
	dest.FeeCharged = int64(tx.Result.Result.FeeCharged)

	lb := hal.LinkBuilder{Base: baseUrl}
	dest.PT = strconv.FormatUint(uint64(tx.TOID()), 10)
	dest.Links.Account = lb.Link("/accounts", dest.Account)
	dest.Links.Ledger = lb.Link("/ledgers", fmt.Sprint(dest.Ledger))
	dest.Links.Operations = lb.PagedLink("/transactions", dest.ID, "operations")
	dest.Links.Effects = lb.PagedLink("/transactions", dest.ID, "effects")
	dest.Links.Self = lb.Link("/transactions", dest.ID)
	dest.Links.Transaction = dest.Links.Self
	dest.Links.Succeeds = lb.Linkf("/transactions?order=desc&cursor=%s", dest.PT)
	dest.Links.Precedes = lb.Linkf("/transactions?order=asc&cursor=%s", dest.PT)

	// If we didn't need the structure, drop it.
	if !tx.HasPreconditions() {
		dest.Preconditions = nil
	}

	return
}

func formatSigners(s []xdr.SignerKey) []string {
	if s == nil {
		return nil
	}

	signers := make([]string, len(s))
	for i, key := range s {
		signers[i] = key.Address()
	}
	return signers
}

func signatures(xdrSignatures []xdr.DecoratedSignature) []string {
	signatures := make([]string, len(xdrSignatures))
	for i, sig := range xdrSignatures {
		signatures[i] = base64.StdEncoding.EncodeToString(sig.Signature)
	}
	return signatures
}

func memoType(transaction archive.LedgerTransaction) string {
	switch transaction.Envelope.Memo().Type {
	case xdr.MemoTypeMemoNone:
		return "none"
	case xdr.MemoTypeMemoText:
		return "text"
	case xdr.MemoTypeMemoId:
		return "id"
	case xdr.MemoTypeMemoHash:
		return "hash"
	case xdr.MemoTypeMemoReturn:
		return "return"
	default:
		panic(fmt.Errorf("invalid memo type: %v", transaction.Envelope.Memo().Type))
	}
}

func memo(transaction archive.LedgerTransaction) (value string, valid bool) {
	valid = true
	memo := transaction.Envelope.Memo()

	switch memo.Type {
	case xdr.MemoTypeMemoNone:
		value, valid = "", false

	case xdr.MemoTypeMemoText:
		scrubbed := scrub(memo.MustText())
		notnull := strings.Join(strings.Split(scrubbed, "\x00"), "")
		value = notnull

	case xdr.MemoTypeMemoId:
		value = fmt.Sprintf("%d", memo.MustId())

	case xdr.MemoTypeMemoHash:
		hash := memo.MustHash()
		value = base64.StdEncoding.EncodeToString(hash[:])

	case xdr.MemoTypeMemoReturn:
		hash := memo.MustRetHash()
		value = base64.StdEncoding.EncodeToString(hash[:])

	default:
		panic(fmt.Errorf("invalid memo type: %v", memo.Type))
	}

	return
}

func memoBytes(envelopeXDR string) (string, error) {
	var parsedEnvelope xdr.TransactionEnvelope
	if err := xdr.SafeUnmarshalBase64(envelopeXDR, &parsedEnvelope); err != nil {
		return "", err
	}

	memo := *parsedEnvelope.Memo().Text
	return base64.StdEncoding.EncodeToString([]byte(memo)), nil
}

// scrub ensures that a given string is valid utf-8, replacing any invalid byte
// sequences with the utf-8 replacement character.
func scrub(in string) string {
	// First check validity using the stdlib, returning if the string is already
	// valid
	if utf8.ValidString(in) {
		return in
	}

	left := []byte(in)
	var result bytes.Buffer

	for len(left) > 0 {
		r, n := utf8.DecodeRune(left)
		result.WriteRune(r) // never errors, only panics
		left = left[n:]
	}

	return result.String()
}

func formatTime[T constraints.Integer](t T) string {
	return strconv.FormatUint(uint64(t), 10)
}
