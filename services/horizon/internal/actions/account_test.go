package actions

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

var (
	trustLineIssuer = "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H"
	accountOne      = "GABGMPEKKDWR2WFH5AJOZV5PDKLJEHGCR3Q24ALETWR5H3A7GI3YTS7V"
	accountTwo      = "GADTXHUTHIAESMMQ2ZWSTIIGBZRLHUCBLCHPLLUEIAWDEFRDC4SYDKOZ"
	accountThree    = "GDP347UYM2ZKE6ED6T5OM3BQ5IAS76NKRVEUPNB5PCQ26Z5D7Q7PJOMI"
	signer          = "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU"
	usd             = xdr.MustNewCreditAsset("USD", trustLineIssuer)
	euro            = xdr.MustNewCreditAsset("EUR", trustLineIssuer)

	account1 = xdr.AccountEntry{
		AccountId:     xdr.MustAddress(accountOne),
		Balance:       20000,
		SeqNum:        223456789,
		NumSubEntries: 10,
		Flags:         1,
		HomeDomain:    "stellar.org",
		Thresholds:    xdr.Thresholds{1, 2, 3, 4},
		Ext: xdr.AccountEntryExt{
			V: 1,
			V1: &xdr.AccountEntryV1{
				Liabilities: xdr.Liabilities{
					Buying:  3,
					Selling: 4,
				},
			},
		},
	}

	account2 = xdr.AccountEntry{
		AccountId:     xdr.MustAddress(accountTwo),
		Balance:       50000,
		SeqNum:        648736,
		NumSubEntries: 10,
		Flags:         2,
		HomeDomain:    "meridian.stellar.org",
		Thresholds:    xdr.Thresholds{5, 6, 7, 8},
		Ext: xdr.AccountEntryExt{
			V: 1,
			V1: &xdr.AccountEntryV1{
				Liabilities: xdr.Liabilities{
					Buying:  30,
					Selling: 40,
				},
			},
		},
	}

	account3 = xdr.AccountEntry{
		AccountId:     xdr.MustAddress(signer),
		Balance:       50000,
		SeqNum:        648736,
		NumSubEntries: 10,
		Flags:         2,
		Thresholds:    xdr.Thresholds{5, 6, 7, 8},
		Ext: xdr.AccountEntryExt{
			V: 1,
			V1: &xdr.AccountEntryV1{
				Liabilities: xdr.Liabilities{
					Buying:  30,
					Selling: 40,
				},
			},
		},
	}

	eurTrustLine = xdr.TrustLineEntry{
		AccountId: xdr.MustAddress(accountOne),
		Asset:     euro,
		Balance:   20000,
		Limit:     223456789,
		Flags:     1,
		Ext: xdr.TrustLineEntryExt{
			V: 1,
			V1: &xdr.TrustLineEntryV1{
				Liabilities: xdr.Liabilities{
					Buying:  3,
					Selling: 4,
				},
			},
		},
	}

	usdTrustLine = xdr.TrustLineEntry{
		AccountId: xdr.MustAddress(accountTwo),
		Asset:     usd,
		Balance:   10000,
		Limit:     123456789,
		Flags:     0,
		Ext: xdr.TrustLineEntryExt{
			V: 1,
			V1: &xdr.TrustLineEntryV1{
				Liabilities: xdr.Liabilities{
					Buying:  1,
					Selling: 2,
				},
			},
		},
	}

	data1 = xdr.DataEntry{
		AccountId: xdr.MustAddress(accountOne),
		DataName:  "test data",
		// This also tests if base64 encoding is working as 0 is invalid UTF-8 byte
		DataValue: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	}

	data2 = xdr.DataEntry{
		AccountId: xdr.MustAddress(accountTwo),
		DataName:  "test data2",
		DataValue: []byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
	}

	accountSigners = []history.AccountSigner{
		history.AccountSigner{
			Account: accountOne,
			Signer:  accountOne,
			Weight:  1,
		},
		history.AccountSigner{
			Account: accountTwo,
			Signer:  accountTwo,
			Weight:  1,
		},
		history.AccountSigner{
			Account: accountOne,
			Signer:  signer,
			Weight:  1,
		},
		history.AccountSigner{
			Account: accountTwo,
			Signer:  signer,
			Weight:  2,
		},
		history.AccountSigner{
			Account: signer,
			Signer:  signer,
			Weight:  3,
		},
	}
)

func TestAccountInfo(t *testing.T) {
	tt := test.Start(t).Scenario("allow_trust")
	defer tt.Finish()

	account, err := AccountInfo(
		tt.Ctx,
		&core.Q{tt.CoreSession()},
		&history.Q{tt.HorizonSession()},
		signer,
		false,
	)
	tt.Assert.NoError(err)

	tt.Assert.Equal("8589934593", account.Sequence)
	tt.Assert.NotEqual(0, account.LastModifiedLedger)

	for _, balance := range account.Balances {
		if balance.Type == "native" {
			tt.Assert.Equal(uint32(0), balance.LastModifiedLedger)
		} else {
			tt.Assert.NotEqual(uint32(0), balance.LastModifiedLedger)
		}
	}
}
func TestGetAccountsHandlerPageNoResults(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &history.Q{tt.HorizonSession()}
	handler := &GetAccountsHandler{}
	records, err := handler.GetResourcePage(
		httptest.NewRecorder(),
		makeRequest(
			t,
			map[string]string{
				"signer": signer,
			},
			map[string]string{},
			q.Session,
		),
	)
	tt.Assert.NoError(err)
	tt.Assert.Len(records, 0)
}

func TestGetAccountsHandlerPageResultsBySigner(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &history.Q{tt.HorizonSession()}
	handler := &GetAccountsHandler{}

	_, err := q.InsertAccount(account1, 1234)
	tt.Assert.NoError(err)
	_, err = q.InsertAccount(account2, 1234)
	tt.Assert.NoError(err)
	_, err = q.InsertAccount(account3, 1234)
	tt.Assert.NoError(err)

	for _, row := range accountSigners {
		q.CreateAccountSigner(row.Account, row.Signer, row.Weight)
	}

	records, err := handler.GetResourcePage(
		httptest.NewRecorder(),
		makeRequest(
			t,
			map[string]string{
				"signer": signer,
			},
			map[string]string{},
			q.Session,
		),
	)

	tt.Assert.NoError(err)
	tt.Assert.Equal(3, len(records))

	want := map[string]bool{
		accountOne: true,
		accountTwo: true,
		signer:     true,
	}

	for _, row := range records {
		result := row.(protocol.Account)
		tt.Assert.True(want[result.AccountID])
		delete(want, result.AccountID)
	}

	tt.Assert.Empty(want)

	records, err = handler.GetResourcePage(
		httptest.NewRecorder(),
		makeRequest(
			t,
			map[string]string{
				"signer": signer,
				"cursor": accountOne,
			},
			map[string]string{},
			q.Session,
		),
	)

	tt.Assert.NoError(err)
	tt.Assert.Equal(2, len(records))

	want = map[string]bool{
		accountTwo: true,
		signer:     true,
	}

	for _, row := range records {
		result := row.(protocol.Account)
		tt.Assert.True(want[result.AccountID])
		delete(want, result.AccountID)
	}

	tt.Assert.Empty(want)
}

func TestGetAccountsHandlerPageResultsByAsset(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &history.Q{tt.HorizonSession()}
	handler := &GetAccountsHandler{}

	_, err := q.InsertAccount(account1, 1234)
	tt.Assert.NoError(err)
	_, err = q.InsertAccount(account2, 1234)
	tt.Assert.NoError(err)

	for _, row := range accountSigners {
		_, err = q.CreateAccountSigner(row.Account, row.Signer, row.Weight)
		tt.Assert.NoError(err)
	}

	_, err = q.InsertAccountData(data1, 1234)
	assert.NoError(t, err)
	_, err = q.InsertAccountData(data2, 1234)
	assert.NoError(t, err)

	var assetType, code, issuer string
	usd.MustExtract(&assetType, &code, &issuer)
	params := map[string]string{
		"asset": code + ":" + issuer,
	}

	records, err := handler.GetResourcePage(
		httptest.NewRecorder(),
		makeRequest(
			t,
			params,
			map[string]string{},
			q.Session,
		),
	)

	tt.Assert.NoError(err)
	tt.Assert.Equal(0, len(records))

	_, err = q.InsertTrustLine(eurTrustLine, 1234)
	assert.NoError(t, err)
	_, err = q.InsertTrustLine(usdTrustLine, 1235)
	assert.NoError(t, err)

	records, err = handler.GetResourcePage(
		httptest.NewRecorder(),
		makeRequest(
			t,
			params,
			map[string]string{},
			q.Session,
		),
	)

	tt.Assert.NoError(err)
	tt.Assert.Equal(1, len(records))
	result := records[0].(protocol.Account)
	tt.Assert.Equal(accountTwo, result.AccountID)
	tt.Assert.Len(result.Balances, 2)
	tt.Assert.Len(result.Signers, 2)

	_, ok := result.Data[string(data2.DataName)]
	tt.Assert.True(ok)
}

func TestGetAccountsHandlerInvalidParams(t *testing.T) {
	testCases := []struct {
		desc                    string
		params                  map[string]string
		expectedInvalidField    string
		expectedErr             string
		isInvalidAccountsParams bool
	}{
		{
			desc:                    "empty filters",
			isInvalidAccountsParams: true,
		},
		{
			desc: "signer and seller",
			params: map[string]string{
				"signer": accountOne,
				"asset":  "USD" + ":" + accountOne,
			},
			expectedInvalidField: "signer",
			expectedErr:          "you can't filter by signer and asset at the same time",
		},
		{
			desc: "filtering by native asset",
			params: map[string]string{
				"asset": "native",
			},
			expectedInvalidField: "asset",
			expectedErr:          "you can't filter by asset: native",
		},
		{
			desc: "invalid asset",
			params: map[string]string{
				"asset_issuer": accountOne,
				"asset":        "USDCOP:someissuer",
			},
			expectedInvalidField: "asset",
			expectedErr:          customTagsErrorMessages["asset"],
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tt := test.Start(t)
			defer tt.Finish()
			q := &history.Q{tt.HorizonSession()}
			handler := &GetAccountsHandler{}

			_, err := handler.GetResourcePage(
				httptest.NewRecorder(),
				makeRequest(
					t,
					tc.params,
					map[string]string{},
					q.Session,
				),
			)
			tt.Assert.Error(err)
			if tc.isInvalidAccountsParams {
				tt.Assert.Equal(invalidAccountsParams, err)
			} else {
				if tt.Assert.IsType(&problem.P{}, err) {
					p := err.(*problem.P)
					tt.Assert.Equal("bad_request", p.Type)
					tt.Assert.Equal(tc.expectedInvalidField, p.Extras["invalid_field"])
					tt.Assert.Equal(
						tc.expectedErr,
						p.Extras["reason"],
					)
				}
			}
		})
	}
}

func TestAccountQueryURLTemplate(t *testing.T) {
	tt := assert.New(t)
	expected := "/accounts{?signer,asset,cursor,limit,order}"
	accountsQuery := AccountsQuery{}
	tt.Equal(expected, accountsQuery.URITemplate())
}
