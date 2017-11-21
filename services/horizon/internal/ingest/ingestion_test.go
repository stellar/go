package ingest

import (
	"testing"

	"github.com/stellar/go/support/db"
	"github.com/stellar/go/xdr"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	testDB "github.com/stellar/go/services/horizon/internal/test/db"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stretchr/testify/assert"
)

func TestEmptySignature(t *testing.T) {
	ingestion := Ingestion{
		DB: &db.Session{
			DB: testDB.Horizon(t),
		},
	}
	ingestion.Start()

	envelope := xdr.TransactionEnvelope{}
	resultPair := xdr.TransactionResultPair{}
	meta := xdr.TransactionMeta{}

	xdr.SafeUnmarshalBase64("AAAAAMIK9djC7k75ziKOLJcvMAIBG7tnBuoeI34x+Pi6zqcZAAAAZAAZphYAAAABAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAynnCTTyw53VVRLOWX6XKTva63IM1LslPNW01YB0hz/8AAAAAAAAAAlQL5AAAAAAAAAAAAh0hz/8AAABA8qkkeKaKfsbgInyIkzXJhqJE5/Ufxri2LdxmyKkgkT6I3sPmvrs5cPWQSzEQyhV750IW2ds97xTHqTpOfuZCAnhSuFUAAAAA", &envelope)
	xdr.SafeUnmarshalBase64("AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=", &resultPair.Result)
	xdr.SafeUnmarshalBase64("AAAAAAAAAAEAAAADAAAAAQAZphoAAAAAAAAAAMIK9djC7k75ziKOLJcvMAIBG7tnBuoeI34x+Pi6zqcZAAAAF0h255wAGaYWAAAAAQAAAAMAAAAAAAAAAAAAAAADBQUFAAAAAwAAAAAtkqVYLPLYhqNMmQLPc+T9eTWp8LIE8eFlR5K4wNJKTQAAAAMAAAAAynnCTTyw53VVRLOWX6XKTva63IM1LslPNW01YB0hz/8AAAADAAAAAuOwxEKY/BwUmvv0yJlvuSQnrkHkZJuTTKSVmRt4UrhVAAAAAwAAAAAAAAAAAAAAAwAZphYAAAAAAAAAAMp5wk08sOd1VUSzll+lyk72utyDNS7JTzVtNWAdIc//AAAAF0h26AAAGaYWAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAZphoAAAAAAAAAAMp5wk08sOd1VUSzll+lyk72utyDNS7JTzVtNWAdIc//AAAAGZyCzAAAGaYWAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA", &meta)

	transaction := &core.Transaction{
		TransactionHash: "1939a8de30981e4171e1aaeca54a058a7fb06684864facba0620ab8cc5076d4f",
		LedgerSequence:  1680922,
		Index:           1,
		Envelope:        envelope,
		Result:          resultPair,
		ResultMeta:      meta,
	}

	transactionFee := &core.TransactionFee{}

	builder := ingestion.transactionInsertBuilder(1, transaction, transactionFee)
	sql, args, err := builder.ToSql()
	assert.Equal(t, "INSERT INTO history_transactions (id,transaction_hash,ledger_sequence,application_order,account,account_sequence,fee_paid,operation_count,tx_envelope,tx_result,tx_meta,tx_fee_meta,signatures,time_bounds,memo_type,memo,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?::character varying[],?,?,?,?,?)", sql)
	assert.Equal(t, `{"8qkkeKaKfsbgInyIkzXJhqJE5/Ufxri2LdxmyKkgkT6I3sPmvrs5cPWQSzEQyhV750IW2ds97xTHqTpOfuZCAg==",""}`, args[12])
	assert.NoError(t, err)

	err = ingestion.Transaction(1, transaction, transactionFee)
	assert.NoError(t, err)

	err = ingestion.Close()
	assert.NoError(t, err)
}

func TestAssetIngest(t *testing.T) {
	//ingest kahuna and sample a single expected asset output

	tt := test.Start(t).ScenarioWithoutHorizon("kahuna")
	defer tt.Finish()
	s := ingest(tt)
	q := history.Q{Session: s.Ingestion.DB}

	expectedAsset := history.Asset{
		ID     : 4,
		Type   : "credit_alphanum4",
		Code   : "USD",
		Issuer : "GB2QIYT2IAUFMRXKLSLLPRECC6OCOGJMADSPTRK7TGNT2SFR2YGWDARD",
	}

	actualAsset := history.Asset{}
	err := q.GetAssetByID(&actualAsset, 4)
	tt.Require.NoError(err)
	tt.Assert.Equal(expectedAsset, actualAsset)
}

func TestTradeIngestTimestamp(t *testing.T) {
	//ingest trade scenario and verify that the trade timestamp
	//matches the appropriate ledger's timestamp
	tt := test.Start(t).ScenarioWithoutHorizon("trades")
	defer tt.Finish()
	s := ingest(tt)
	q := history.Q{Session: s.Ingestion.DB}

	var ledgers []history.Ledger
	err := q.Ledgers().Select(&ledgers)
	tt.Require.NoError(err)

	var trades []history.Trade
	err = q.Trades().Select(&trades)
	tt.Require.NoError(err)

	tt.Require.Equal(trades[len(trades)-1].LedgerCloseTime, ledgers[len(ledgers)-1].ClosedAt)
}
