package resourceadapter

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
	"time"

	"github.com/jackc/pgtype"
	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/httpx"
	"github.com/stellar/go/support/render/hal"
)

// Populate fills out the details
func PopulateTransaction(
	ctx context.Context,
	transactionHash string,
	dest *protocol.Transaction,
	row history.Transaction,
) error {
	dest.ID = transactionHash
	dest.PT = row.PagingToken()
	dest.Successful = row.Successful
	dest.Hash = transactionHash
	dest.Ledger = row.LedgerSequence
	dest.LedgerCloseTime = row.LedgerCloseTime
	dest.Account = row.Account
	dest.AccountSequence = row.AccountSequence

	dest.FeeCharged = row.FeeCharged

	dest.OperationCount = row.OperationCount
	dest.EnvelopeXdr = row.TxEnvelope
	dest.ResultXdr = row.TxResult
	dest.ResultMetaXdr = row.TxMeta
	dest.FeeMetaXdr = row.TxFeeMeta
	dest.MemoType = row.MemoType
	dest.Memo = row.Memo.String
	if row.MemoType == "text" {
		if memoBytes, err := memoBytes(row.TxEnvelope); err != nil {
			return err
		} else {
			dest.MemoBytes = memoBytes
		}
	}
	if err := row.Signatures.AssignTo(&dest.Signatures); err != nil {
		return errors.Wrap(err, "could not parse signatures")
	}
	dest.ValidBefore = timeString(dest, row.TimeBounds.Upper)
	dest.ValidAfter = timeString(dest, row.TimeBounds.Lower)

	if row.InnerTransactionHash.Valid {
		dest.FeeAccount = row.FeeAccount.String
		dest.MaxFee = row.NewMaxFee.Int64
		dest.FeeBumpTransaction = &protocol.FeeBumpTransaction{
			Hash:       row.TransactionHash,
			Signatures: dest.Signatures,
		}
		dest.InnerTransaction = &protocol.InnerTransaction{
			Hash:   row.InnerTransactionHash.String,
			MaxFee: row.MaxFee,
		}
		if err := row.InnerSignatures.AssignTo(&dest.InnerTransaction.Signatures); err != nil {
			return errors.Wrap(err, "could not parse inner signatures")
		}
		if transactionHash != row.TransactionHash {
			dest.Signatures = dest.InnerTransaction.Signatures
		}
	} else {
		dest.FeeAccount = row.Account
		dest.MaxFee = row.MaxFee
	}

	lb := hal.LinkBuilder{Base: httpx.BaseURL(ctx)}
	dest.Links.Account = lb.Link("/accounts", dest.Account)
	dest.Links.Ledger = lb.Link("/ledgers", fmt.Sprintf("%d", dest.Ledger))
	dest.Links.Operations = lb.PagedLink("/transactions", dest.ID, "operations")
	dest.Links.Effects = lb.PagedLink("/transactions", dest.ID, "effects")
	dest.Links.Self = lb.Link("/transactions", dest.ID)
	dest.Links.Transaction = dest.Links.Self
	dest.Links.Succeeds = lb.Linkf("/transactions?order=desc&cursor=%s", dest.PT)
	dest.Links.Precedes = lb.Linkf("/transactions?order=asc&cursor=%s", dest.PT)

	return nil
}

func memoBytes(envelopeXDR string) (string, error) {
	var parsedEnvelope xdr.TransactionEnvelope
	if err := xdr.SafeUnmarshalBase64(envelopeXDR, &parsedEnvelope); err != nil {
		return "", err
	}

	memo := *parsedEnvelope.Memo().Text
	return base64.StdEncoding.EncodeToString([]byte(memo)), nil
}

func timeString(res *protocol.Transaction, in pgtype.Int8) string {
	if in.Status == pgtype.Null {
		return ""
	}

	return time.Unix(in.Int, 0).UTC().Format(time.RFC3339)
}
