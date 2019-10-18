package actions

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

func TestAssetStatsValidation(t *testing.T) {
	handler := assetStatsHandler{}

	for _, testCase := range []struct {
		name               string
		queryParams        map[string]string
		expectedErrorField string
		expectedError      string
	}{
		{
			"invalid asset code",
			map[string]string{
				"asset_code": "tooooooooolong",
			},
			"asset_code",
			"not a valid asset code",
		},
		{
			"invalid asset issuer",
			map[string]string{
				"asset_issuer": "invalid",
			},
			"asset_issuer",
			"not a valid asset issuer",
		},
		{
			"cursor has too many colons",
			map[string]string{
				"cursor": "ABC:GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H:",
			},
			"cursor",
			"cursor must contain exactly one colon",
		},
		{
			"invalid cursor code",
			map[string]string{
				"cursor": "tooooooooolong:GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
			},
			"cursor",
			"not a valid asset code",
		},
		{
			"invalid cursor issuer",
			map[string]string{
				"cursor": "ABC:invalidissuer",
			},
			"cursor",
			"not a valid asset issuer",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			r := makeRequest(t, testCase.queryParams, map[string]string{})
			_, err := handler.GetResourcePage(httptest.NewRecorder(), r)
			if err == nil {
				t.Fatalf("expected error %v but got %v", testCase.expectedError, err)
			}

			problem := err.(*problem.P)
			if field := problem.Extras["invalid_field"]; field != testCase.expectedErrorField {
				t.Fatalf(
					"expected error field %v but got %v",
					testCase.expectedErrorField,
					field,
				)
			}

			reason := problem.Extras["reason"]
			if !strings.Contains(reason.(string), testCase.expectedError) {
				t.Fatalf("expected reason %v but got %v", testCase.expectedError, reason)
			}
		})
	}
}

func TestAssetStats(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &history.Q{tt.HorizonSession()}
	handler := NewAssetStats(q)

	issuer := history.AccountEntry{
		AccountID: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		Flags: uint32(xdr.AccountFlagsAuthRequiredFlag) |
			uint32(xdr.AccountFlagsAuthImmutableFlag),
	}
	issuerFlags := horizon.AccountFlags{
		AuthRequired:  true,
		AuthImmutable: true,
	}
	otherIssuer := history.AccountEntry{
		AccountID:  "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2",
		HomeDomain: "xim.com",
	}

	usdAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: issuer.AccountID,
		AssetCode:   "USD",
		Amount:      "1",
		NumAccounts: 2,
	}
	usdAssetStatResponse := horizon.AssetStat{
		Amount:      "0.0000001",
		NumAccounts: usdAssetStat.NumAccounts,
		Asset: base.Asset{
			Type:   "credit_alphanum4",
			Code:   usdAssetStat.AssetCode,
			Issuer: usdAssetStat.AssetIssuer,
		},
		PT:    usdAssetStat.AssetCode + ":" + usdAssetStat.AssetIssuer,
		Flags: issuerFlags,
	}

	etherAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: issuer.AccountID,
		AssetCode:   "ETHER",
		Amount:      "23",
		NumAccounts: 1,
	}
	etherAssetStatResponse := horizon.AssetStat{
		Amount:      "0.0000023",
		NumAccounts: etherAssetStat.NumAccounts,
		Asset: base.Asset{
			Type:   "credit_alphanum4",
			Code:   etherAssetStat.AssetCode,
			Issuer: etherAssetStat.AssetIssuer,
		},
		PT:    etherAssetStat.AssetCode + ":" + etherAssetStat.AssetIssuer,
		Flags: issuerFlags,
	}

	otherUSDAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: otherIssuer.AccountID,
		AssetCode:   "USD",
		Amount:      "1",
		NumAccounts: 2,
	}
	otherUSDAssetStatResponse := horizon.AssetStat{
		Amount:      "0.0000001",
		NumAccounts: otherUSDAssetStat.NumAccounts,
		Asset: base.Asset{
			Type:   "credit_alphanum4",
			Code:   otherUSDAssetStat.AssetCode,
			Issuer: otherUSDAssetStat.AssetIssuer,
		},
		PT: otherUSDAssetStat.AssetCode + ":" + otherUSDAssetStat.AssetIssuer,
	}
	otherUSDAssetStatResponse.Links.Toml = hal.NewLink(
		"https://" + otherIssuer.HomeDomain + "/.well-known/stellar.toml",
	)

	eurAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: otherIssuer.AccountID,
		AssetCode:   "EUR",
		Amount:      "111",
		NumAccounts: 3,
	}
	eurAssetStatResponse := horizon.AssetStat{
		Amount:      "0.0000111",
		NumAccounts: eurAssetStat.NumAccounts,
		Asset: base.Asset{
			Type:   "credit_alphanum4",
			Code:   eurAssetStat.AssetCode,
			Issuer: eurAssetStat.AssetIssuer,
		},
		PT: eurAssetStat.AssetCode + ":" + eurAssetStat.AssetIssuer,
	}
	eurAssetStatResponse.Links.Toml = hal.NewLink(
		"https://" + otherIssuer.HomeDomain + "/.well-known/stellar.toml",
	)

	for _, assetStat := range []history.ExpAssetStat{
		etherAssetStat,
		eurAssetStat,
		otherUSDAssetStat,
		usdAssetStat,
	} {
		numChanged, err := q.InsertAssetStat(assetStat)
		tt.Assert.NoError(err)
		tt.Assert.Equal(numChanged, int64(1))
	}

	for _, account := range []history.AccountEntry{
		issuer,
		otherIssuer,
	} {
		accountEntry := xdr.AccountEntry{
			Flags:      xdr.Uint32(account.Flags),
			HomeDomain: xdr.String32(account.HomeDomain),
		}
		if err := accountEntry.AccountId.SetAddress(account.AccountID); err != nil {
			t.Fatalf("unexpected error %v", err)
		}
		numChanged, err := q.InsertAccount(accountEntry, 3)
		tt.Assert.NoError(err)
		tt.Assert.Equal(numChanged, int64(1))
	}

	for _, testCase := range []struct {
		name        string
		queryParams map[string]string
		expected    []horizon.AssetStat
	}{
		{
			"default parameters",
			map[string]string{},
			[]horizon.AssetStat{
				etherAssetStatResponse,
				eurAssetStatResponse,
				otherUSDAssetStatResponse,
				usdAssetStatResponse,
			},
		},
		{
			"with cursor",
			map[string]string{
				"cursor": etherAssetStatResponse.PagingToken(),
			},
			[]horizon.AssetStat{
				eurAssetStatResponse,
				otherUSDAssetStatResponse,
				usdAssetStatResponse,
			},
		},
		{
			"descending order",
			map[string]string{"order": "desc"},
			[]horizon.AssetStat{
				usdAssetStatResponse,
				otherUSDAssetStatResponse,
				eurAssetStatResponse,
				etherAssetStatResponse,
			},
		},
		{
			"filter by asset code",
			map[string]string{
				"asset_code": "USD",
			},
			[]horizon.AssetStat{
				otherUSDAssetStatResponse,
				usdAssetStatResponse,
			},
		},
		{
			"filter by asset issuer",
			map[string]string{
				"asset_issuer": issuer.AccountID,
			},
			[]horizon.AssetStat{
				etherAssetStatResponse,
				usdAssetStatResponse,
			},
		},
		{
			"filter by both asset code and asset issuer",
			map[string]string{
				"asset_code":   "USD",
				"asset_issuer": issuer.AccountID,
			},
			[]horizon.AssetStat{
				usdAssetStatResponse,
			},
		},
		{
			"filter produces empty set",
			map[string]string{
				"asset_code":   "XYZ",
				"asset_issuer": issuer.AccountID,
			},
			[]horizon.AssetStat{},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			r := makeRequest(t, testCase.queryParams, map[string]string{})
			results, err := handler.GetResourcePage(httptest.NewRecorder(), r)
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}

			if len(results) != len(testCase.expected) {
				t.Fatalf(
					"expectes results to have length %v but got %v",
					len(results),
					len(testCase.expected),
				)
			}

			for i, item := range results {
				assetStat := item.(horizon.AssetStat)
				if assetStat != testCase.expected[i] {
					t.Fatalf("expected %v but got %v", testCase.expected[i], assetStat)
				}
			}
		})
	}
}

func TestAssetStatsIssuerDoesNotExist(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &history.Q{tt.HorizonSession()}
	handler := NewAssetStats(q)

	usdAssetStat := history.ExpAssetStat{
		AssetType:   xdr.AssetTypeAssetTypeCreditAlphanum4,
		AssetIssuer: "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		AssetCode:   "USD",
		Amount:      "1",
		NumAccounts: 2,
	}
	numChanged, err := q.InsertAssetStat(usdAssetStat)
	tt.Assert.NoError(err)
	tt.Assert.Equal(numChanged, int64(1))

	r := makeRequest(t, map[string]string{}, map[string]string{})
	_, err = handler.GetResourcePage(httptest.NewRecorder(), r)
	if err == nil {
		t.Fatal("error but got not nil")
	}

	expected := "Account for issuer GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H does not exist"

	if err.Error() != expected {
		t.Fatalf("expected error %v but got %v", expected, err)
	}
}
