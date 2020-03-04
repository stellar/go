package horizon

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/expingest"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

var (
	data1 = xdr.DataEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		DataName:  "name1",
		// This also tests if base64 encoding is working as 0 is invalid UTF-8 byte
		DataValue: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	}

	data2 = xdr.DataEntry{
		AccountId: xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"),
		DataName:  "name ",
		DataValue: []byte("it got spaces!"),
	}
)

func TestDataActions_Show(t *testing.T) {
	ht := StartHTTPTestWithoutScenario(t)
	defer ht.Finish()
	test.ResetHorizonDB(t, ht.HorizonDB)
	q := &history.Q{ht.HorizonSession()}

	// Makes StateMiddleware happy
	err := q.UpdateLastLedgerExpIngest(100)
	ht.Assert.NoError(err)
	err = q.UpdateExpIngestVersion(expingest.CurrentVersion)
	ht.Assert.NoError(err)
	_, err = q.InsertLedger(xdr.LedgerHeaderHistoryEntry{
		Header: xdr.LedgerHeader{
			LedgerSeq: 100,
		},
	}, 0, 0, 0, 0)
	ht.Assert.NoError(err)

	rows, err := q.InsertAccountData(data1, 1234)
	assert.NoError(t, err)
	ht.Assert.Equal(int64(1), rows)

	rows, err = q.InsertAccountData(data2, 1235)
	assert.NoError(t, err)
	ht.Assert.Equal(int64(1), rows)

	prefix := "/accounts/GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"
	var result map[string]string

	// json
	w := ht.Get(prefix + "/data/name1")
	if ht.Assert.Equal(200, w.Code) {
		err := json.Unmarshal(w.Body.Bytes(), &result)
		ht.Assert.NoError(err)
		decoded, err := base64.StdEncoding.DecodeString(result["value"])
		ht.Assert.NoError(err)
		ht.Assert.Equal([]byte(data1.DataValue), decoded)
	}

	// raw
	w = ht.Get(prefix+"/data/name1", test.RequestHelperRaw)
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.Equal([]byte(data1.DataValue), w.Body.Bytes())
	}

	// regression: https://github.com/stellar/horizon/issues/325
	// names with special characters do not work
	w = ht.Get(prefix + "/data/name%20")
	if ht.Assert.Equal(200, w.Code) {
		err := json.Unmarshal(w.Body.Bytes(), &result)
		ht.Assert.NoError(err)

		decoded, err := base64.StdEncoding.DecodeString(result["value"])
		ht.Assert.NoError(err)
		ht.Assert.Equal([]byte(data2.DataValue), decoded)
	}

	w = ht.Get(prefix+"/data/name%20", test.RequestHelperRaw)
	if ht.Assert.Equal(200, w.Code) {
		ht.Assert.Equal("it got spaces!", w.Body.String())
	}

	// missing
	w = ht.Get(prefix + "/data/missing")
	ht.Assert.Equal(404, w.Code)

	w = ht.Get(prefix+"/data/missing", test.RequestHelperRaw)
	ht.Assert.Equal(404, w.Code)
}
