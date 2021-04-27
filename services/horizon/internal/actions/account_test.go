package actions

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	protocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

var (
	trustLineIssuer = "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H"
	accountOne      = "GABGMPEKKDWR2WFH5AJOZV5PDKLJEHGCR3Q24ALETWR5H3A7GI3YTS7V"
	accountTwo      = "GADTXHUTHIAESMMQ2ZWSTIIGBZRLHUCBLCHPLLUEIAWDEFRDC4SYDKOZ"
	signer          = "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU"
	sponsor         = xdr.MustAddress("GCO26ZSBD63TKYX45H2C7D2WOFWOUSG5BMTNC3BG4QMXM3PAYI6WHKVZ")
	usd             = xdr.MustNewCreditAsset("USD", trustLineIssuer)
	euro            = xdr.MustNewCreditAsset("EUR", trustLineIssuer)

	account1 = xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1234,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeAccount,
			Account: &xdr.AccountEntry{
				AccountId:     xdr.MustAddress(accountOne),
				Balance:       20000,
				SeqNum:        223456789,
				NumSubEntries: 10,
				Flags:         1,
				HomeDomain:    "stellar.org",
				Thresholds:    xdr.Thresholds{1, 2, 3, 4},
				Ext: xdr.AccountEntryExt{
					V: 1,
					V1: &xdr.AccountEntryExtensionV1{
						Liabilities: xdr.Liabilities{
							Buying:  3,
							Selling: 4,
						},
					},
				},
			},
		},
	}

	account2 = xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1234,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeAccount,
			Account: &xdr.AccountEntry{
				AccountId:     xdr.MustAddress(accountTwo),
				Balance:       50000,
				SeqNum:        648736,
				NumSubEntries: 10,
				Flags:         2,
				HomeDomain:    "meridian.stellar.org",
				Thresholds:    xdr.Thresholds{5, 6, 7, 8},
				Ext: xdr.AccountEntryExt{
					V: 1,
					V1: &xdr.AccountEntryExtensionV1{
						Liabilities: xdr.Liabilities{
							Buying:  30,
							Selling: 40,
						},
					},
				},
			},
		},
	}

	account3 = xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1234,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeAccount,
			Account: &xdr.AccountEntry{
				AccountId:     xdr.MustAddress(signer),
				Balance:       50000,
				SeqNum:        648736,
				NumSubEntries: 10,
				Flags:         2,
				Thresholds:    xdr.Thresholds{5, 6, 7, 8},
				Ext: xdr.AccountEntryExt{
					V: 1,
					V1: &xdr.AccountEntryExtensionV1{
						Liabilities: xdr.Liabilities{
							Buying:  30,
							Selling: 40,
						},
						Ext: xdr.AccountEntryExtensionV1Ext{
							V2: &xdr.AccountEntryExtensionV2{
								NumSponsored:  1,
								NumSponsoring: 2,
							},
						},
					},
				},
			},
		},
		Ext: xdr.LedgerEntryExt{
			V: 1,
			V1: &xdr.LedgerEntryExtensionV1{
				SponsoringId: &sponsor,
			},
		},
	}

	eurTrustLine = xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1234,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeTrustline,
			TrustLine: &xdr.TrustLineEntry{
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
			},
		},
	}

	usdTrustLine = xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1234,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeTrustline,
			TrustLine: &xdr.TrustLineEntry{
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
			},
		},
	}

	data1 = xdr.LedgerEntry{
		LastModifiedLedgerSeq: 100,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeData,
			Data: &xdr.DataEntry{
				AccountId: xdr.MustAddress(accountOne),
				DataName:  "test data",
				// This also tests if base64 encoding is working as 0 is invalid UTF-8 byte
				DataValue: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			},
		},
	}

	data2 = xdr.LedgerEntry{
		LastModifiedLedgerSeq: 100,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeData,
			Data: &xdr.DataEntry{
				AccountId: xdr.MustAddress(accountTwo),
				DataName:  "test data2",
				DataValue: []byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
			},
		},
	}

	accountSigners = []history.AccountSigner{
		{
			Account: accountOne,
			Signer:  accountOne,
			Weight:  1,
		},
		{
			Account: accountTwo,
			Signer:  accountTwo,
			Weight:  1,
		},
		{
			Account: accountOne,
			Signer:  signer,
			Weight:  1,
		},
		{
			Account: accountTwo,
			Signer:  signer,
			Weight:  2,
		},
		{
			Account: signer,
			Signer:  signer,
			Weight:  3,
		},
	}
)

func TestAccountInfo(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &history.Q{tt.HorizonSession()}

	var thresholds xdr.Thresholds
	tt.Assert.NoError(
		xdr.SafeUnmarshalBase64("AQAAAA==", &thresholds),
	)
	accountID := xdr.MustAddress(
		"GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU",
	)
	accountEntry := xdr.LedgerEntry{
		LastModifiedLedgerSeq: 4,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeAccount,
			Account: &xdr.AccountEntry{
				AccountId:     accountID,
				Balance:       9999999900,
				SeqNum:        8589934593,
				NumSubEntries: 1,
				InflationDest: nil,
				HomeDomain:    "",
				Thresholds:    thresholds,
				Flags:         0,
			},
		},
	}
	batch := q.NewAccountsBatchInsertBuilder(0)
	err := batch.Add(tt.Ctx, accountEntry)
	assert.NoError(t, err)
	assert.NoError(t, batch.Exec(tt.Ctx))

	tt.Assert.NoError(err)

	_, err = q.InsertTrustLine(tt.Ctx, xdr.LedgerEntry{
		LastModifiedLedgerSeq: 6,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeTrustline,
			TrustLine: &xdr.TrustLineEntry{
				AccountId: accountID,
				Asset: xdr.MustNewCreditAsset(
					"USD",
					"GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
				),
				Balance: 0,
				Limit:   9223372036854775807,
				Flags:   1,
			},
		},
	})
	assert.NoError(t, err)

	_, err = q.InsertTrustLine(tt.Ctx, xdr.LedgerEntry{
		LastModifiedLedgerSeq: 1234,
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeTrustline,
			TrustLine: &xdr.TrustLineEntry{
				AccountId: accountID,
				Asset: xdr.MustNewCreditAsset(
					"EUR",
					"GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
				),
				Balance: 0,
				Limit:   9223372036854775807,
				Flags:   1,
			},
		},
	})
	assert.NoError(t, err)

	ledgerFourCloseTime := time.Now().Unix()
	_, err = q.InsertLedger(tt.Ctx, xdr.LedgerHeaderHistoryEntry{
		Header: xdr.LedgerHeader{
			LedgerSeq: 4,
			ScpValue: xdr.StellarValue{
				CloseTime: xdr.TimePoint(ledgerFourCloseTime),
			},
		},
	}, 0, 0, 0, 0, 0)
	assert.NoError(t, err)

	account, err := AccountInfo(tt.Ctx, &history.Q{tt.HorizonSession()}, accountID.Address())
	tt.Assert.NoError(err)

	tt.Assert.Equal("8589934593", account.Sequence)
	tt.Assert.Equal(uint32(4), account.LastModifiedLedger)
	tt.Assert.NotNil(account.LastModifiedTime)
	tt.Assert.Equal(ledgerFourCloseTime, account.LastModifiedTime.Unix())
	tt.Assert.Len(account.Balances, 3)

	tt.Assert.Equal(account.Balances[0].Code, "EUR")
	tt.Assert.Equal(account.Balances[1].Code, "USD")
	tt.Assert.Equal(
		"GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4",
		account.Balances[1].Issuer,
	)
	tt.Assert.NotEqual(uint32(0), account.Balances[1].LastModifiedLedger)
	tt.Assert.Equal(account.Balances[2].Type, "native")
	tt.Assert.Equal(uint32(0), account.Balances[2].LastModifiedLedger)
	tt.Assert.Len(account.Signers, 1)

	// Regression: no trades link
	tt.Assert.Contains(account.Links.Trades.Href, "/trades")
	// Regression: no data link
	tt.Assert.Contains(account.Links.Data.Href, "/data/{key}")
	tt.Assert.True(account.Links.Data.Templated)

	// try to fetch account which does not exist
	_, err = AccountInfo(tt.Ctx, &history.Q{tt.HorizonSession()}, "GDBAPLDCAEJV6LSEDFEAUDAVFYSNFRUYZ4X75YYJJMMX5KFVUOHX46SQ")
	tt.Assert.True(q.NoRows(errors.Cause(err)))
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

	batch := q.NewAccountsBatchInsertBuilder(0)
	err := batch.Add(tt.Ctx, account1)
	assert.NoError(t, err)
	err = batch.Add(tt.Ctx, account2)
	assert.NoError(t, err)
	err = batch.Add(tt.Ctx, account3)
	assert.NoError(t, err)
	assert.NoError(t, batch.Exec(tt.Ctx))

	for _, row := range accountSigners {
		q.CreateAccountSigner(tt.Ctx, row.Account, row.Signer, row.Weight, nil)
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

func TestGetAccountsHandlerPageResultsBySponsor(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &history.Q{tt.HorizonSession()}
	handler := &GetAccountsHandler{}

	batch := q.NewAccountsBatchInsertBuilder(0)
	err := batch.Add(tt.Ctx, account1)
	assert.NoError(t, err)
	err = batch.Add(tt.Ctx, account2)
	assert.NoError(t, err)
	err = batch.Add(tt.Ctx, account3)
	assert.NoError(t, err)
	assert.NoError(t, batch.Exec(tt.Ctx))

	for _, row := range accountSigners {
		q.CreateAccountSigner(tt.Ctx, row.Account, row.Signer, row.Weight, nil)
	}

	records, err := handler.GetResourcePage(
		httptest.NewRecorder(),
		makeRequest(
			t,
			map[string]string{
				"sponsor": sponsor.Address(),
			},
			map[string]string{},
			q.Session,
		),
	)

	tt.Assert.NoError(err)
	tt.Assert.Equal(1, len(records))
	tt.Assert.Equal(signer, records[0].(protocol.Account).ID)
}

func TestGetAccountsHandlerPageResultsByAsset(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)

	q := &history.Q{tt.HorizonSession()}
	handler := &GetAccountsHandler{}

	batch := q.NewAccountsBatchInsertBuilder(0)
	err := batch.Add(tt.Ctx, account1)
	assert.NoError(t, err)
	err = batch.Add(tt.Ctx, account2)
	assert.NoError(t, err)
	assert.NoError(t, batch.Exec(tt.Ctx))
	ledgerCloseTime := time.Now().Unix()
	_, err = q.InsertLedger(tt.Ctx, xdr.LedgerHeaderHistoryEntry{
		Header: xdr.LedgerHeader{
			LedgerSeq: 1234,
			ScpValue: xdr.StellarValue{
				CloseTime: xdr.TimePoint(ledgerCloseTime),
			},
		},
	}, 0, 0, 0, 0, 0)
	assert.NoError(t, err)

	for _, row := range accountSigners {
		_, err = q.CreateAccountSigner(tt.Ctx, row.Account, row.Signer, row.Weight, nil)
		tt.Assert.NoError(err)
	}

	_, err = q.InsertAccountData(tt.Ctx, data1)
	assert.NoError(t, err)
	_, err = q.InsertAccountData(tt.Ctx, data2)
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

	_, err = q.InsertTrustLine(tt.Ctx, eurTrustLine)
	assert.NoError(t, err)
	_, err = q.InsertTrustLine(tt.Ctx, usdTrustLine)
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
	tt.Assert.NotNil(result.LastModifiedTime)
	tt.Assert.Equal(ledgerCloseTime, result.LastModifiedTime.Unix())
	tt.Assert.Len(result.Balances, 2)
	tt.Assert.Len(result.Signers, 2)

	_, ok := result.Data[string(data2.Data.Data.DataName)]
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
			isInvalidAccountsParams: true,
		},
		{
			desc: "signer and sponsor",
			params: map[string]string{
				"signer":  accountOne,
				"sponsor": accountTwo,
			},
			isInvalidAccountsParams: true,
		},
		{
			desc: "asset and sponsor",
			params: map[string]string{
				"asset":   "USD" + ":" + accountOne,
				"sponsor": accountTwo,
			},
			isInvalidAccountsParams: true,
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
	expected := "/accounts{?signer,sponsor,asset,cursor,limit,order}"
	accountsQuery := AccountsQuery{}
	tt.Equal(expected, accountsQuery.URITemplate())
}
