package ingest

import (
	"testing"

	protocolEffects "github.com/stellar/go/protocols/horizon/effects"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

func Test_ingestSignerEffects(t *testing.T) {
	tt := test.Start(t).ScenarioWithoutHorizon("set_options")
	defer tt.Finish()

	s := ingest(tt, Config{EnableAssetStats: false})
	tt.Require.NoError(s.Err)

	q := &history.Q{Session: tt.HorizonSession()}

	// Regression: https://github.com/stellar/horizon/issues/390 doesn't produce a signer effect when
	// inflation has changed
	var effects []history.Effect
	err := q.Effects().ForLedger(3).Select(&effects)
	tt.Require.NoError(err)

	if tt.Assert.Len(effects, 1) {
		tt.Assert.NotEqual(history.EffectSignerUpdated, effects[0].Type)
	}
}

func Test_ingestOperationEffects(t *testing.T) {
	tt := test.Start(t).ScenarioWithoutHorizon("set_options")
	defer tt.Finish()

	s := ingest(tt, Config{EnableAssetStats: false})
	tt.Require.NoError(s.Err)

	q := &history.Q{Session: tt.HorizonSession()}
	var effects []history.Effect

	// ensure inflation destination change is correctly recorded
	err := q.Effects().ForLedger(3).Select(&effects)
	tt.Require.NoError(err)

	if tt.Assert.Len(effects, 1) {
		tt.Assert.Equal(history.EffectAccountInflationDestinationUpdated, effects[0].Type)
	}

	// HACK(scott): switch to kahuna recipe mid-stream.  We need to integrate our test scenario loader to be compatible with go subtests/
	tt.ScenarioWithoutHorizon("kahuna")
	s = ingest(tt, Config{EnableAssetStats: false})
	tt.Require.NoError(s.Err)
	pq, err := db2.NewPageQuery("", true, "asc", 200)
	tt.Require.NoError(err)

	// ensure payments get the payment effects
	err = q.Effects().ForLedger(15).Page(pq).Select(&effects)
	tt.Require.NoError(err)

	if tt.Assert.Len(effects, 2) {
		tt.Assert.Equal(history.EffectAccountCredited, effects[0].Type)
		tt.Assert.Equal(history.EffectAccountDebited, effects[1].Type)
	}

	// ensure path payments get the payment effects
	err = q.Effects().ForLedger(20).Page(pq).Select(&effects)
	tt.Require.NoError(err)

	if tt.Assert.Len(effects, 4) {
		tt.Assert.Equal(history.EffectAccountCredited, effects[0].Type)
		tt.Assert.Equal(history.EffectAccountDebited, effects[1].Type)
		tt.Assert.Equal(history.EffectTrade, effects[2].Type)
		tt.Assert.Equal(history.EffectTrade, effects[3].Type)
	}

	err = q.Effects().ForOperation(81604382721).Page(pq).Select(&effects)
	tt.Require.NoError(err)

	var ad protocolEffects.AccountDebited
	err = effects[1].UnmarshalDetails(&ad)
	tt.Require.NoError(err)
	tt.Assert.Equal("100.0000000", ad.Amount)
}

func Test_ingestBumpSeq(t *testing.T) {
	tt := test.Start(t).ScenarioWithoutHorizon("kahuna")
	defer tt.Finish()

	s := ingest(tt, Config{EnableAssetStats: false})
	tt.Require.NoError(s.Err)

	q := &history.Q{Session: tt.HorizonSession()}

	//ensure bumpseq operations
	ops, _, err := q.Operations().ForAccount("GCQZP3IU7XU6EJ63JZXKCQOYT2RNXN3HB5CNHENNUEUHSMA4VUJJJSEN").Fetch()
	tt.Require.NoError(err)
	if tt.Assert.Len(ops, 5) {
		//first is create account, and then bump sequences
		tt.Assert.Equal(xdr.OperationTypeCreateAccount, ops[0].Type)
		for i := 1; i < 5; i++ {
			tt.Assert.Equal(xdr.OperationTypeBumpSequence, ops[i].Type)
		}
	}

	//ensure bumpseq effect
	var effects []history.Effect
	err = q.Effects().OfType(history.EffectSequenceBumped).Select(&effects)
	tt.Require.NoError(err)

	//sample a bumpseq effect
	if tt.Assert.Len(effects, 1) {
		testEffect := effects[0]
		details := struct {
			NewSq int64 `json:"new_seq"`
		}{}
		err = testEffect.UnmarshalDetails(&details)
		tt.Assert.Equal(int64(300000000000), details.NewSq)
	}
}

func Test_ingestPathPaymentStrictSend(t *testing.T) {
	tt := test.Start(t).ScenarioWithoutHorizon("paths_strict_send")
	defer tt.Finish()

	s := ingest(tt, Config{EnableAssetStats: false})
	tt.Require.NoError(s.Err)

	q := &history.Q{Session: tt.HorizonSession()}

	ops, _, err := q.Operations().ForAccount("GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU").Fetch()
	tt.Require.NoError(err)
	if tt.Assert.Len(ops, 5) {
		tt.Assert.Equal(xdr.OperationTypeCreateAccount, ops[0].Type)
		tt.Assert.Equal(xdr.OperationTypeChangeTrust, ops[1].Type)
		tt.Assert.Equal(xdr.OperationTypePayment, ops[2].Type)
		for i := 3; i < 5; i++ {
			tt.Assert.Equal(xdr.OperationTypePathPaymentStrictSend, ops[i].Type)
		}
	}

	var effects []history.Effect
	err = q.Effects().ForAccount("GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU").OfType(history.EffectAccountDebited).Select(&effects)
	tt.Require.NoError(err)

	details := struct {
		Amount      string `json:"amount"`
		AssetType   string `json:"asset_type"`
		AssetCode   string `json:"asset_code"`
		AssetIssuer string `json:"asset_issuer"`
	}{}

	if tt.Assert.Len(effects, 2) {
		err = effects[0].UnmarshalDetails(&details)
		tt.Assert.NoError(err)
		tt.Assert.Equal("10.0000000", details.Amount)
		tt.Assert.Equal("credit_alphanum4", details.AssetType)
		tt.Assert.Equal("USD", details.AssetCode)
		tt.Assert.Equal("GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", details.AssetIssuer)

		err = effects[1].UnmarshalDetails(&details)
		tt.Assert.NoError(err)
		tt.Assert.Equal("12.0000000", details.Amount)
		tt.Assert.Equal("credit_alphanum4", details.AssetType)
		tt.Assert.Equal("USD", details.AssetCode)
		tt.Assert.Equal("GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", details.AssetIssuer)
	}

	err = q.Effects().ForAccount("GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2").OfType(history.EffectAccountCredited).Select(&effects)
	tt.Require.NoError(err)

	if tt.Assert.Len(effects, 3) {
		// effects[0] is simple payment

		err = effects[1].UnmarshalDetails(&details)
		tt.Assert.NoError(err)
		tt.Assert.Equal("13.0000000", details.Amount)
		tt.Assert.Equal("credit_alphanum4", details.AssetType)
		tt.Assert.Equal("EUR", details.AssetCode)
		tt.Assert.Equal("GCQPYGH4K57XBDENKKX55KDTWOTK5WDWRQOH2LHEDX3EKVIQRLMESGBG", details.AssetIssuer)

		err = effects[2].UnmarshalDetails(&details)
		tt.Assert.NoError(err)
		tt.Assert.Equal("15.8400000", details.Amount)
		tt.Assert.Equal("credit_alphanum4", details.AssetType)
		tt.Assert.Equal("EUR", details.AssetCode)
		tt.Assert.Equal("GCQPYGH4K57XBDENKKX55KDTWOTK5WDWRQOH2LHEDX3EKVIQRLMESGBG", details.AssetIssuer)
	}

	err = q.Effects().ForAccount("GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU").OfType(history.EffectTrade).Select(&effects)
	tt.Require.NoError(err)

	tradeDetails := struct {
		OfferID           int64  `json:"offer_id"`
		Seller            string `json:"seller"`
		BoughtAmount      string `json:"bought_amount"`
		BoughtAssetType   string `json:"bought_asset_type"`
		BoughtAssetCode   string `json:"bought_asset_code"`
		BoughtAssetIssuer string `json:"bought_asset_issuer"`
		SoldAmount        string `json:"sold_amount"`
		SoldAssetType     string `json:"sold_asset_type"`
		SoldAssetCode     string `json:"sold_asset_code"`
		SoldAssetIssuer   string `json:"sold_asset_issuer"`
	}{}

	if tt.Assert.Len(effects, 3) {
		err = effects[0].UnmarshalDetails(&tradeDetails)
		tt.Assert.NoError(err)
		tt.Assert.Equal(int64(3), tradeDetails.OfferID)
		tt.Assert.Equal("GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON", tradeDetails.Seller)
		tt.Assert.Equal("13.0000000", tradeDetails.BoughtAmount)
		tt.Assert.Equal("credit_alphanum4", tradeDetails.BoughtAssetType)
		tt.Assert.Equal("EUR", tradeDetails.BoughtAssetCode)
		tt.Assert.Equal("GCQPYGH4K57XBDENKKX55KDTWOTK5WDWRQOH2LHEDX3EKVIQRLMESGBG", tradeDetails.BoughtAssetIssuer)
		tt.Assert.Equal("10.0000000", tradeDetails.SoldAmount)
		tt.Assert.Equal("credit_alphanum4", tradeDetails.SoldAssetType)
		tt.Assert.Equal("USD", tradeDetails.SoldAssetCode)
		tt.Assert.Equal("GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", tradeDetails.SoldAssetIssuer)

		err = effects[1].UnmarshalDetails(&tradeDetails)
		tt.Assert.NoError(err)
		tt.Assert.Equal(int64(1), tradeDetails.OfferID)
		tt.Assert.Equal("GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON", tradeDetails.Seller)
		tt.Assert.Equal("13.2000000", tradeDetails.BoughtAmount)
		tt.Assert.Equal("native", tradeDetails.BoughtAssetType)
		tt.Assert.Equal("12.0000000", tradeDetails.SoldAmount)
		tt.Assert.Equal("credit_alphanum4", tradeDetails.SoldAssetType)
		tt.Assert.Equal("USD", tradeDetails.SoldAssetCode)
		tt.Assert.Equal("GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", tradeDetails.SoldAssetIssuer)

		err = effects[2].UnmarshalDetails(&tradeDetails)
		tt.Assert.NoError(err)
		tt.Assert.Equal(int64(2), tradeDetails.OfferID)
		tt.Assert.Equal("GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON", tradeDetails.Seller)
		tt.Assert.Equal("15.8400000", tradeDetails.BoughtAmount)
		tt.Assert.Equal("credit_alphanum4", tradeDetails.BoughtAssetType)
		tt.Assert.Equal("EUR", tradeDetails.BoughtAssetCode)
		tt.Assert.Equal("GCQPYGH4K57XBDENKKX55KDTWOTK5WDWRQOH2LHEDX3EKVIQRLMESGBG", tradeDetails.BoughtAssetIssuer)
		tt.Assert.Equal("13.2000000", tradeDetails.SoldAmount)
		tt.Assert.Equal("native", tradeDetails.SoldAssetType)
	}
}

func Test_ingestPathPaymentStrictSendTxTooLate(t *testing.T) {
	tt := test.Start(t).ScenarioWithoutHorizon("paths_strict_send")
	defer tt.Finish()

	_, err := tt.CoreSession().ExecRaw(
		`UPDATE txhistory SET txresult = 'aJIrokUPctlM/8KI4lsLiIdqmy/f6fa9J/4xMMbtF54AAAAAAAAAZP////0AAAAA' WHERE txid = ?`,
		"0a1bb4fc8e39ac99730cc36326c0289621956a6f9d2e92ee927d762a670840cc",
	)
	tt.Require.NoError(err)

	s := ingest(tt, Config{EnableAssetStats: false, IngestFailedTransactions: true})
	tt.Require.NoError(s.Err)

	q := &history.Q{Session: tt.HorizonSession()}
	ops, _, err := q.Operations().ForAccount("GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU").IncludeFailed().Fetch()
	tt.Require.NoError(err)
	if tt.Assert.Len(ops, 6) {
		tt.Assert.Equal(xdr.OperationTypeCreateAccount, ops[0].Type)
		tt.Assert.Equal(xdr.OperationTypeChangeTrust, ops[1].Type)
		tt.Assert.Equal(xdr.OperationTypePayment, ops[2].Type)
		for i := 3; i < 6; i++ {
			tt.Assert.Equal(xdr.OperationTypePathPaymentStrictSend, ops[i].Type)
		}
	}

	details := struct {
		Amount         string `json:"amount"`
		DestinationMin string `json:"destination_min"`
	}{}

	err = ops[5].UnmarshalDetails(&details)
	tt.Require.NoError(err)
	tt.Assert.Equal("0.0000000", details.Amount)
	tt.Assert.Equal("100.0000000", details.DestinationMin)
}
