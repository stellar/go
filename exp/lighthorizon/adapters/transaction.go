package adapters

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/stellar/go/exp/lighthorizon/common"
	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/render/hal"
)

func PopulateTransaction(r *http.Request, tx *common.Transaction) (protocol.Transaction, error) {
	hash, err := tx.TransactionHash()
	if err != nil {
		return protocol.Transaction{}, err
	}

	toid := strconv.FormatInt(tx.TOID(), 10)
	envelopeXdr, err := tx.TransactionEnvelope.MarshalBinary()
	if err != nil {
		return protocol.Transaction{}, err
	}
	resultXdr, err := tx.TransactionResult.MarshalBinary()
	if err != nil {
		return protocol.Transaction{}, err
	}
	// resultMetaXdr, err := tx.TransactionResultMeta.MarshalBinary()
	// if err != nil {
	// 	return protocol.Transaction{}, err
	// }
	// feeMetaXdr, err := tx.TransactionResult.Me
	memo := tx.TransactionEnvelope.Memo()
	base := protocol.Transaction{
		ID:              hash,
		PT:              toid,
		Successful:      tx.TransactionResult.Successful(),
		Hash:            hash,
		Ledger:          int32(tx.LedgerHeader.LedgerSeq),
		LedgerCloseTime: time.Unix(int64(tx.LedgerHeader.ScpValue.CloseTime), 0).UTC(),
		Account:         tx.SourceAccount().ToAccountId().Address(),
		FeeCharged:      int64(tx.TransactionResult.FeeCharged),
		OperationCount:  int32(len(tx.TransactionEnvelope.Operations())),
		EnvelopeXdr:     base64.URLEncoding.EncodeToString(envelopeXdr),
		ResultXdr:       base64.URLEncoding.EncodeToString(resultXdr),
		// ResultMetaXdr:   string(resultMetaXdr),
		// FeeMetaXdr: string(feeMetaXdr),
		MemoType: memo.Type.String(),
		Memo:     memo.GoString(),
	}
	// TODO: Fill in the rest of the fields

	// timebounds := tx.TransactionEnvelope.TimeBounds()
	// if timebounds != nil {
	// 	base.ValidBefore = timeString(&base, &timebounds.MaxTime)
	// 	base.ValidAfter = timeString(&base, &timebounds.MinTime)
	// }

	for _, sig := range tx.TransactionEnvelope.Signatures() {
		b, err := sig.MarshalBinary()
		if err != nil {
			return protocol.Transaction{}, err
		}
		base.Signatures = append(base.Signatures, base64.URLEncoding.EncodeToString(b))
	}

	// if memo.Type.String() == "text" {
	// 	if memoBytes, err := memoBytes(row.TxEnvelope); err != nil {
	// 		return err
	// 	} else {
	// 		base.MemoBytes = memoBytes
	// 	}
	// }

	lb := hal.LinkBuilder{Base: r.URL}
	self := fmt.Sprintf("/transactions/%s", hash)
	base.Links.Self = lb.Link(self)
	base.Links.Succeeds = lb.Linkf("/effects?order=desc&cursor=%s", base.PT)
	base.Links.Precedes = lb.Linkf("/effects?order=asc&cursor=%s", base.PT)
	base.Links.Effects = lb.Link(self, "effects")
	return base, nil
}
