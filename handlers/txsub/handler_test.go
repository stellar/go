package txsub

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/support/http/httptest"
	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {
	// happy path with limit
	// Mock the upstream horizon
	hmock := httptest.NewClient()
	client := &horizon.Client{
		URL:  "https://localhost",
		HTTP: hmock,
	}

	// Mock the horizon proxy transaction submission service
	proxymock := httptest.NewClient()
	driver := InitHorizonProxyDriverMock(*client, *proxymock)
	handler := &Handler{
		Driver:  driver,
		Ticks:   time.NewTicker(1 * time.Second),
		Context: context.Background(),
	}

	server := httptest.NewServer(t, handler)
	defer server.Close()

	go handler.Run()

	// Delay transaction "confirmation"
	go func(hmock *httptest.Client) {
		time.Sleep(2 * time.Second)
		fmt.Println("Changed")
		hmock.On(
			"GET",
			fmt.Sprintf("https://localhost/transactions/cb323b02148e231570c3573b7a563dfea0c8cdb1c15cdd5aaf04acf4ce4b702a"),
		).ReturnString(200, transactionResponse)
	}(hmock)

	hmock.On(
		"POST",
		fmt.Sprintf("https://localhost/transactions"),
	).ReturnString(200, transactionResponse)

	hmock.On(
		"GET",
		"https://localhost/accounts/GBCHJCAATUZPVNOMRGQ7GJOMLB7IEMNSVCKADFKHNTHQLHU2GOJKUMDW",
	).ReturnString(200, accountResponse)

	z := server.POST("/tx").
		WithFormField("tx", tx).
		Expect().
		Status(http.StatusOK).
		ContentType("application/hal+json").
		Body().Raw()

	assert.Equal(t, z, expectedResponseSuccess)
	fmt.Println("Passed happy path")

	// Transaction Already submitted
	// Mock the upstream horizon
	hmock = httptest.NewClient()
	client = &horizon.Client{
		URL:  "https://localhost",
		HTTP: hmock,
	}

	// Mock the horizon proxy transaction submission service
	proxymock = httptest.NewClient()
	driver = InitHorizonProxyDriverMock(*client, *proxymock)
	handler = &Handler{
		Driver:  driver,
		Ticks:   time.NewTicker(1 * time.Second),
		Context: context.Background(),
	}

	server = httptest.NewServer(t, handler)
	defer server.Close()

	go handler.Run()

	hmock.On(
		"POST",
		fmt.Sprintf("https://localhost/transactions"),
	).ReturnString(200, transactionResponse)

	hmock.On(
		"GET",
		"https://localhost/accounts/GBCHJCAATUZPVNOMRGQ7GJOMLB7IEMNSVCKADFKHNTHQLHU2GOJKUMDW",
	).ReturnString(200, accountResponse)

	hmock.On(
		"GET",
		fmt.Sprintf("https://localhost/transactions/cb323b02148e231570c3573b7a563dfea0c8cdb1c15cdd5aaf04acf4ce4b702a"),
	).ReturnString(200, transactionResponse)

	z = server.POST("/tx").
		WithFormField("tx", tx).
		Expect().
		Status(http.StatusOK).
		ContentType("application/hal+json").
		Body().Raw()

	assert.Equal(t, z, expectedResponseSuccess)
	fmt.Println("Passed already submitted")

	// Transaction Bad Sequence
	// Mock the upstream horizon
	hmock = httptest.NewClient()
	client = &horizon.Client{
		URL:  "https://localhost",
		HTTP: hmock,
	}

	// Mock the horizon proxy transaction submission service
	proxymock = httptest.NewClient()
	driver = InitHorizonProxyDriverMock(*client, *proxymock)
	handler = &Handler{
		Driver:  driver,
		Ticks:   time.NewTicker(1 * time.Second),
		Context: context.Background(),
	}

	server = httptest.NewServer(t, handler)
	defer server.Close()

	go handler.Run()

	hmock.On(
		"POST",
		fmt.Sprintf("https://localhost/transactions"),
	).ReturnString(200, transactionResponse)

	hmock.On(
		"GET",
		"https://localhost/accounts/GBCHJCAATUZPVNOMRGQ7GJOMLB7IEMNSVCKADFKHNTHQLHU2GOJKUMDW",
	).ReturnString(200, accountResponse2)

	z = server.POST("/tx").
		WithFormField("tx", tx).
		Expect().
		Status(http.StatusOK).
		ContentType("application/hal+json").
		Body().Raw()

	assert.Equal(t, z, expectedResponseBadSequence)
	fmt.Println("Passed bad sequence")

	// Transaction Timedout
	// Mock the upstream horizon
	hmock = httptest.NewClient()
	client = &horizon.Client{
		URL:  "https://localhost",
		HTTP: hmock,
	}

	// Mock the horizon proxy transaction submission service
	proxymock = httptest.NewClient()
	driver = InitHorizonProxyDriverMock(*client, *proxymock)
	handler = &Handler{
		Driver:  driver,
		Ticks:   time.NewTicker(1 * time.Second),
		Context: context.Background(),
	}

	server = httptest.NewServer(t, handler)
	defer server.Close()

	go handler.Run()

	hmock.On(
		"POST",
		fmt.Sprintf("https://localhost/transactions"),
	).ReturnString(200, transactionResponse)

	hmock.On(
		"GET",
		"https://localhost/accounts/GBCHJCAATUZPVNOMRGQ7GJOMLB7IEMNSVCKADFKHNTHQLHU2GOJKUMDW",
	).ReturnString(200, accountResponse)

	z = server.POST("/tx").
		WithFormField("tx", tx).
		Expect().
		Status(http.StatusOK).
		ContentType("application/hal+json").
		Body().Raw()

	assert.Equal(t, z, expectedResponseTimeout)
	fmt.Println("Passed timed out")

}

var tx = "AAAAAER0iACdMvq1zImh8yXMWH6CMbKolAGVR2zPBZ6aM5KqAAAAZACUWO8AAAACAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAARHSIAJ0y+rXMiaHzJcxYfoIxsqiUAZVHbM8FnpozkqoAAAAAAAAAAAX14QAAAAAAAAAAAZozkqoAAABA3kweRZ9OTHXS4r7uRjbOUCu/7uOHkqIp5/dIVhCGeqzlDQJXqaLICt441Nj+C40dyDigTlQfmrZ3NLZXDR+nAQ=="

var accountResponse = `{
	"_links": {
	  "self": {
		"href": "https://horizon-testnet.stellar.org/accounts/GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H"
	  },
	  "transactions": {
		"href": "https://horizon-testnet.stellar.org/accounts/GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H/transactions{?cursor,limit,order}",
		"templated": true
	  },
	  "operations": {
		"href": "https://horizon-testnet.stellar.org/accounts/GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H/operations{?cursor,limit,order}",
		"templated": true
	  },
	  "payments": {
		"href": "https://horizon-testnet.stellar.org/accounts/GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H/payments{?cursor,limit,order}",
		"templated": true
	  },
	  "effects": {
		"href": "https://horizon-testnet.stellar.org/accounts/GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H/effects{?cursor,limit,order}",
		"templated": true
	  },
	  "offers": {
		"href": "https://horizon-testnet.stellar.org/accounts/GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H/Offers{?cursor,limit,order}",
		"templated": true
	  }
	},
	"id": "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
	"paging_token": "1",
	"account_id": "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
	"sequence": "41756080073605121",
	"subentry_count": 0,
	"thresholds": {
	  "low_threshold": 0,
	  "med_threshold": 0,
	  "high_threshold": 0
	},
	"flags": {
	  "auth_required": false,
	  "auth_revocable": false
	},
	"balances": [
	  {
		"balance": "948522307.6146000",
		"asset_type": "native"
	  }
	],
	"signers": [
	  {
		"public_key": "XBT5HNPK6DAL6222MAWTLHNOZSDKPJ2AKNEQ5Q324CHHCNQFQ7EHBHZN",
		"weight": 1,
		"key": "XBT5HNPK6DAL6222MAWTLHNOZSDKPJ2AKNEQ5Q324CHHCNQFQ7EHBHZN",
		"type": "sha256_hash"
	  },
	  {
		"public_key": "GDQHKHMFW5ICTQYM3QWCXMSZ56BNHMQG6NH6SGV3ZNZ72KRHYV5XINCE",
		"weight": 1,
		"key": "GDQHKHMFW5ICTQYM3QWCXMSZ56BNHMQG6NH6SGV3ZNZ72KRHYV5XINCE",
		"type": "ed25519_public_key"
	  }
	],
	"data": {
	  "test": "R0NCVkwzU1FGRVZLUkxQNkFKNDdVS0tXWUVCWTQ1V0hBSkhDRVpLVldNVEdNQ1Q0SDROS1FZTEg="
	}
  }`

var accountResponse2 = `{
	"_links": {
	  "self": {
		"href": "https://horizon-testnet.stellar.org/accounts/GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H"
	  },
	  "transactions": {
		"href": "https://horizon-testnet.stellar.org/accounts/GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H/transactions{?cursor,limit,order}",
		"templated": true
	  },
	  "operations": {
		"href": "https://horizon-testnet.stellar.org/accounts/GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H/operations{?cursor,limit,order}",
		"templated": true
	  },
	  "payments": {
		"href": "https://horizon-testnet.stellar.org/accounts/GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H/payments{?cursor,limit,order}",
		"templated": true
	  },
	  "effects": {
		"href": "https://horizon-testnet.stellar.org/accounts/GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H/effects{?cursor,limit,order}",
		"templated": true
	  },
	  "offers": {
		"href": "https://horizon-testnet.stellar.org/accounts/GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H/Offers{?cursor,limit,order}",
		"templated": true
	  }
	},
	"id": "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
	"paging_token": "1",
	"account_id": "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
	"sequence": "41756080073605122",
	"subentry_count": 0,
	"thresholds": {
	  "low_threshold": 0,
	  "med_threshold": 0,
	  "high_threshold": 0
	},
	"flags": {
	  "auth_required": false,
	  "auth_revocable": false
	},
	"balances": [
	  {
		"balance": "948522307.6146000",
		"asset_type": "native"
	  }
	],
	"signers": [
	  {
		"public_key": "XBT5HNPK6DAL6222MAWTLHNOZSDKPJ2AKNEQ5Q324CHHCNQFQ7EHBHZN",
		"weight": 1,
		"key": "XBT5HNPK6DAL6222MAWTLHNOZSDKPJ2AKNEQ5Q324CHHCNQFQ7EHBHZN",
		"type": "sha256_hash"
	  },
	  {
		"public_key": "GDQHKHMFW5ICTQYM3QWCXMSZ56BNHMQG6NH6SGV3ZNZ72KRHYV5XINCE",
		"weight": 1,
		"key": "GDQHKHMFW5ICTQYM3QWCXMSZ56BNHMQG6NH6SGV3ZNZ72KRHYV5XINCE",
		"type": "ed25519_public_key"
	  }
	],
	"data": {
	  "test": "R0NCVkwzU1FGRVZLUkxQNkFKNDdVS0tXWUVCWTQ1V0hBSkhDRVpLVldNVEdNQ1Q0SDROS1FZTEg="
	}
  }`

var transactionResponse = `{
	"_links": {
	  "self": {
		"href": "https://horizon.stellar.org/transactions/cb323b02148e231570c3573b7a563dfea0c8cdb1c15cdd5aaf04acf4ce4b702a"
	  },
	  "account": {
		"href": "https://horizon.stellar.org/accounts/GBQ352ACDO6DEGI42SOI4DCB654N7B7DANO4RSBGA5CZLM4475CQNID4"
	  },
	  "ledger": {
		"href": "https://horizon.stellar.org/ledgers/17425656"
	  },
	  "operations": {
		"href": "https://horizon.stellar.org/transactions/cb323b02148e231570c3573b7a563dfea0c8cdb1c15cdd5aaf04acf4ce4b702a/operations{?cursor,limit,order}",
		"templated": true
	  },
	  "effects": {
		"href": "https://horizon.stellar.org/transactions/cb323b02148e231570c3573b7a563dfea0c8cdb1c15cdd5aaf04acf4ce4b702a/effects{?cursor,limit,order}",
		"templated": true
	  },
	  "precedes": {
		"href": "https://horizon.stellar.org/transactions?order=asc&cursor=74842622631374848"
	  },
	  "succeeds": {
		"href": "https://horizon.stellar.org/transactions?order=desc&cursor=74842622631374848"
	  }
	},
	"id": "cb323b02148e231570c3573b7a563dfea0c8cdb1c15cdd5aaf04acf4ce4b702a",
	"paging_token": "74842622631374848",
	"hash": "cb323b02148e231570c3573b7a563dfea0c8cdb1c15cdd5aaf04acf4ce4b702a",
	"ledger": 17425656,
	"created_at": "2018-04-19T00:16:25Z",
	"source_account": "GBQ352ACDO6DEGI42SOI4DCB654N7B7DANO4RSBGA5CZLM4475CQNID4",
	"source_account_sequence": "74842446537687041",
	"fee_paid": 100,
	"operation_count": 1,
	"envelope_xdr": "AAAAAGG+6AIbvDIZHNScjgxB93jfh+MDXcjIJgdFlbOc/0UGAAAAZAEJ5M8AAAABAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAWItna6Yfm4mu3aBouY6Jkq/sVMZZmYKp+Ybebu74C4YAAAAAAJiWgAAAAAAAAAABnP9FBgAAAEB/ufrWJGD1YeVvoxoku9U6CWQTUIO9SGf7NnbZY50Tn7+pNOtNslZy0bYlAabSgoCfJ2ZXRmDMue9v9nrFsLEA",
	"result_xdr": "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=",
	"result_meta_xdr": "AAAAAAAAAAEAAAADAAAAAAEJ5PgAAAAAAAAAAFiLZ2umH5uJrt2gaLmOiZKv7FTGWZmCqfmG3m7u+AuGAAAAAACYloABCeT4AAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAwEJ5PgAAAAAAAAAAGG+6AIbvDIZHNScjgxB93jfh+MDXcjIJgdFlbOc/0UGAAAAAAIWDlwBCeTPAAAAAQAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQEJ5PgAAAAAAAAAAGG+6AIbvDIZHNScjgxB93jfh+MDXcjIJgdFlbOc/0UGAAAAAAF9d9wBCeTPAAAAAQAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA",
	"fee_meta_xdr": "AAAAAgAAAAMBCeT0AAAAAAAAAABhvugCG7wyGRzUnI4MQfd434fjA13IyCYHRZWznP9FBgAAAAACFg7AAQnkzwAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEBCeT4AAAAAAAAAABhvugCG7wyGRzUnI4MQfd434fjA13IyCYHRZWznP9FBgAAAAACFg5cAQnkzwAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==",
	"memo_type": "none",
	"signatures": [
	  "f7n61iRg9WHlb6MaJLvVOglkE1CDvUhn+zZ22WOdE5+/qTTrTbJWctG2JQGm0oKAnydmV0ZgzLnvb/Z6xbCxAA=="
	]
  }`

var expectedResponseSuccess = "{\n  \"Err\": null,\n  \"Hash\": \"cb323b02148e231570c3573b7a563dfea0c8cdb1c15cdd5aaf04acf4ce4b702a\",\n  \"LedgerSequence\": 17425656,\n  \"EnvelopeXDR\": \"AAAAAGG+6AIbvDIZHNScjgxB93jfh+MDXcjIJgdFlbOc/0UGAAAAZAEJ5M8AAAABAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAWItna6Yfm4mu3aBouY6Jkq/sVMZZmYKp+Ybebu74C4YAAAAAAJiWgAAAAAAAAAABnP9FBgAAAEB/ufrWJGD1YeVvoxoku9U6CWQTUIO9SGf7NnbZY50Tn7+pNOtNslZy0bYlAabSgoCfJ2ZXRmDMue9v9nrFsLEA\",\n  \"ResultXDR\": \"AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=\",\n  \"ResultMetaXDR\": \"AAAAAAAAAAEAAAADAAAAAAEJ5PgAAAAAAAAAAFiLZ2umH5uJrt2gaLmOiZKv7FTGWZmCqfmG3m7u+AuGAAAAAACYloABCeT4AAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAwEJ5PgAAAAAAAAAAGG+6AIbvDIZHNScjgxB93jfh+MDXcjIJgdFlbOc/0UGAAAAAAIWDlwBCeTPAAAAAQAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQEJ5PgAAAAAAAAAAGG+6AIbvDIZHNScjgxB93jfh+MDXcjIJgdFlbOc/0UGAAAAAAF9d9wBCeTPAAAAAQAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA\"\n}"
var expectedResponseBadSequence = "{\n  \"Err\": {\n    \"ResultXDR\": \"AAAAAAAAAAD////7AAAAAA==\"\n  },\n  \"Hash\": \"\",\n  \"LedgerSequence\": 0,\n  \"EnvelopeXDR\": \"AAAAAER0iACdMvq1zImh8yXMWH6CMbKolAGVR2zPBZ6aM5KqAAAAZACUWO8AAAACAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAARHSIAJ0y+rXMiaHzJcxYfoIxsqiUAZVHbM8FnpozkqoAAAAAAAAAAAX14QAAAAAAAAAAAZozkqoAAABA3kweRZ9OTHXS4r7uRjbOUCu/7uOHkqIp5/dIVhCGeqzlDQJXqaLICt441Nj+C40dyDigTlQfmrZ3NLZXDR+nAQ==\",\n  \"ResultXDR\": \"\",\n  \"ResultMetaXDR\": \"\"\n}"
var expectedResponseTimeout = "{\n  \"Err\": {},\n  \"Hash\": \"\",\n  \"LedgerSequence\": 0,\n  \"EnvelopeXDR\": \"\",\n  \"ResultXDR\": \"\",\n  \"ResultMetaXDR\": \"\"\n}"
