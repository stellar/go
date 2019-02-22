package core

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
)

func TestLatestLedger(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()
	q := &Q{tt.CoreSession()}

	var seq int
	err := q.LatestLedger(&seq)

	if tt.Assert.NoError(err) {
		tt.Assert.Equal(3, seq)
	}
}

func TestElderLedger(t *testing.T) {
	tt := test.Start(t).ScenarioWithoutHorizon("kahuna")
	defer tt.Finish()
	q := &Q{tt.CoreSession()}

	var elder int32
	err := q.ElderLedger(&elder)
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(elder, int32(1))
	}

	// ledger 3 gets picked properly
	_, err = tt.CoreDB.Exec(`DELETE FROM ledgerheaders WHERE ledgerseq = 2`)
	tt.Require.NoError(err, "failed to remove ledgerheader")

	err = q.ElderLedger(&elder)
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(elder, int32(3))
	}

	// a bigger inital gap is properly dealt with
	_, err = tt.CoreDB.Exec(`
		DELETE FROM ledgerheaders WHERE ledgerseq > 1 AND ledgerseq < 10
	`)
	tt.Require.NoError(err, "failed to remove ledgerheader")

	err = q.ElderLedger(&elder)
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(elder, int32(10))
	}

	// only the latest gap is considered for determining the elder ledger
	_, err = tt.CoreDB.Exec(`
		DELETE FROM ledgerheaders WHERE ledgerseq > 15 AND ledgerseq < 20
	`)
	tt.Require.NoError(err, "failed to remove ledgerheader")

	err = q.ElderLedger(&elder)
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(elder, int32(20))
	}
}

func TestSchemaVersion8(t *testing.T) {
	tt := test.Start(t).ScenarioWithoutHorizon("core_database_schema_version_8")
	defer tt.Finish()
	q := &Q{tt.CoreSession()}

	var account Account
	err := q.AccountByAddress(&account, "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")
	if tt.Assert.NoError(err) {
		tt.Assert.True(account.HomeDomain.Valid)
		tt.Assert.Equal("stellar.org", account.HomeDomain.String)
	}

	var data []AccountData
	err = q.AllDataByAddress(&data, "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(1, len(data))
		tt.Assert.Equal("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H", data[0].Accountid)
		tt.Assert.Equal("aaa", data[0].Key)
		tt.Assert.Equal("bWFu", data[0].Value)
	}

	var singleData AccountData
	err = q.AccountDataByKey(&singleData, "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H", "aaa")
	if tt.Assert.NoError(err) {
		tt.Assert.Equal("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H", singleData.Accountid)
		tt.Assert.Equal("aaa", singleData.Key)
		tt.Assert.Equal("bWFu", singleData.Value)
	}

	var signers []Signer
	err = q.SignersByAddress(&signers, "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(1, len(signers))
		tt.Assert.Equal("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H", signers[0].Accountid)
		tt.Assert.Equal("GAFEES4MDE5Z7Q6JBB2BYMLS7YWEHTPNR7ICANZA7TAOLMSRELE4H4S2", signers[0].Publickey)
		tt.Assert.Equal(int32(2), signers[0].Weight)
	}
}

func TestSchemaVersion9(t *testing.T) {
	tt := test.Start(t).ScenarioWithoutHorizon("core_database_schema_version_9")
	defer tt.Finish()
	q := &Q{tt.CoreSession()}

	var account Account
	err := q.AccountByAddress(&account, "GDZOBPTVEECUYFCHSQ5NCEUVAV4JKRZI6KO5HFOM7HGQT22E3XIGRHNU")
	if tt.Assert.NoError(err) {
		tt.Assert.True(account.HomeDomain.Valid)
		tt.Assert.Equal("lobstr.co", account.HomeDomain.String)
	}

	var data []AccountData
	err = q.AllDataByAddress(&data, "GDZOBPTVEECUYFCHSQ5NCEUVAV4JKRZI6KO5HFOM7HGQT22E3XIGRHNU")
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(1, len(data))
		tt.Assert.Equal("GDZOBPTVEECUYFCHSQ5NCEUVAV4JKRZI6KO5HFOM7HGQT22E3XIGRHNU", data[0].Accountid)
		tt.Assert.Equal("jam", data[0].Key)
		tt.Assert.Equal("bWFu", data[0].Value)
	}

	var singleData AccountData
	err = q.AccountDataByKey(&singleData, "GDZOBPTVEECUYFCHSQ5NCEUVAV4JKRZI6KO5HFOM7HGQT22E3XIGRHNU", "jam")
	if tt.Assert.NoError(err) {
		tt.Assert.Equal("GDZOBPTVEECUYFCHSQ5NCEUVAV4JKRZI6KO5HFOM7HGQT22E3XIGRHNU", singleData.Accountid)
		tt.Assert.Equal("jam", singleData.Key)
		tt.Assert.Equal("bWFu", singleData.Value)
	}

	var signers []Signer
	err = q.SignersByAddress(&signers, "GDZOBPTVEECUYFCHSQ5NCEUVAV4JKRZI6KO5HFOM7HGQT22E3XIGRHNU")
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(1, len(signers))
		tt.Assert.Equal("GDZOBPTVEECUYFCHSQ5NCEUVAV4JKRZI6KO5HFOM7HGQT22E3XIGRHNU", signers[0].Accountid)
		tt.Assert.Equal("GC7BWB2ME4LII3TVWTHUIT7KGJXU4D5M6JUNLQ57WA7JERDNSAEXLOAN", signers[0].Publickey)
		tt.Assert.Equal(int32(10), signers[0].Weight)
	}
}
