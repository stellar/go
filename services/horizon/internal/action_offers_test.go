package horizon

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/expingest"
	"github.com/stellar/go/xdr"
)

func TestOfferActions_Show(t *testing.T) {
	ht := StartHTTPTestWithoutScenario(t)
	defer ht.Finish()
	q := &history.Q{ht.HorizonSession()}

	err := q.UpdateLastLedgerExpIngest(100)
	ht.Assert.NoError(err)
	err = q.UpdateExpIngestVersion(expingest.CurrentVersion)
	ht.Assert.NoError(err)

	ledgerCloseTime := time.Now().Unix()
	_, err = q.InsertLedger(xdr.LedgerHeaderHistoryEntry{
		Header: xdr.LedgerHeader{
			LedgerSeq: 100,
			ScpValue: xdr.StellarValue{
				CloseTime: xdr.TimePoint(ledgerCloseTime),
			},
		},
	}, 0, 0, 0, 0, 0)
	ht.Assert.NoError(err)

	issuer := xdr.MustAddress("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")
	nativeAsset := xdr.MustNewNativeAsset()
	usdAsset := xdr.MustNewCreditAsset("USD", issuer.Address())
	eurAsset := xdr.MustNewCreditAsset("EUR", issuer.Address())

	eurOffer := xdr.LedgerEntry{
		LastModifiedLedgerSeq: 4,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeOffer,
			Offer: &xdr.OfferEntry{
				SellerId: issuer,
				OfferId:  xdr.Int64(4),
				Buying:   eurAsset,
				Selling:  nativeAsset,
				Price: xdr.Price{
					N: 1,
					D: 1,
				},
				Flags:  1,
				Amount: xdr.Int64(500),
			},
		},
	}
	usdOffer := xdr.LedgerEntry{
		LastModifiedLedgerSeq: 3,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeOffer,
			Offer: &xdr.OfferEntry{
				SellerId: issuer,
				OfferId:  xdr.Int64(6),
				Buying:   usdAsset,
				Selling:  eurAsset,
				Price: xdr.Price{
					N: 1,
					D: 1,
				},
				Flags:  1,
				Amount: xdr.Int64(500),
			},
		},
	}

	batch := q.NewOffersBatchInsertBuilder(3)
	err = batch.Add(eurOffer)
	ht.Assert.NoError(err)
	err = batch.Add(usdOffer)
	ht.Assert.NoError(err)
	ht.Assert.NoError(batch.Exec())

	w := ht.Get("/offers")
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.PageOf(2, w.Body)
	}

	w = ht.Get("/offers/4")
	if ht.Assert.Equal(200, w.Code) {
		var response horizon.Offer
		err = json.Unmarshal(w.Body.Bytes(), &response)
		ht.Assert.NoError(err)
		ht.Assert.Equal(int64(4), response.ID)
	}
}
