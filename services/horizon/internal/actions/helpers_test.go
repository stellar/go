package actions

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"

	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/problem"
	"github.com/stellar/go/xdr"
)

func TestGetAccountID(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	action := makeTestAction()

	aid := action.GetAccountID("4_asset_issuer")
	tt.Assert.NoError(action.Err)
	tt.Assert.Equal(
		"GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		aid.Address(),
	)
}

func TestGetTransactionID(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	action := makeTestAction()

	txID, err := GetTransactionID(action.R, "valid_tx_id")
	tt.Assert.NoError(err)
	tt.Assert.Equal(
		"aa168f12124b7c196c0adaee7c73a64d37f99428cacb59a91ff389626845e7cf",
		txID,
	)

	txID, err = GetTransactionID(action.R, "invalid_uppercase_tx_id")
	tt.Assert.Error(err)

	txID, err = GetTransactionID(action.R, "invalid_too_short_tx_id")
	tt.Assert.Error(err)
}

func TestGetAsset(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	action := makeTestAction()

	ts := action.GetAsset("native_")
	if tt.Assert.NoError(action.Err) {
		tt.Assert.Equal(xdr.AssetTypeAssetTypeNative, ts.Type)
	}

	ts = action.GetAsset("4_")
	if tt.Assert.NoError(action.Err) {
		tt.Assert.Equal(xdr.AssetTypeAssetTypeCreditAlphanum4, ts.Type)
	}

	ts = action.GetAsset("12_")
	if tt.Assert.NoError(action.Err) {
		tt.Assert.Equal(xdr.AssetTypeAssetTypeCreditAlphanum12, ts.Type)
	}

	// bad path
	action.GetAsset("cursor")
	tt.Assert.Error(action.Err)

	// regression #298:  GetAsset panics when asset_code is longer than allowes
	tt.Assert.NotPanics(func() {
		action.Err = nil
		action.GetAsset("long_4_")
	})

	tt.Assert.NotPanics(func() {
		action.Err = nil
		action.GetAsset("long_12_")
	})
}

func TestGetAssetType(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	r := makeTestAction().R

	ts, err := getAssetType(r, "native_asset_type")
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(xdr.AssetTypeAssetTypeNative, ts)
	}

	ts, err = getAssetType(r, "4_asset_type")
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(xdr.AssetTypeAssetTypeCreditAlphanum4, ts)
	}

	ts, err = getAssetType(r, "12_asset_type")
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(xdr.AssetTypeAssetTypeCreditAlphanum12, ts)
	}
}

func TestMaybeGetAsset(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	r := makeTestAction().R

	ts, found := MaybeGetAsset(r, "native_")
	if tt.Assert.True(found) {
		tt.Assert.Equal(xdr.AssetTypeAssetTypeNative, ts.Type)
	}

	ts, found = MaybeGetAsset(r, "4_")
	if tt.Assert.True(found) {
		tt.Assert.Equal(xdr.AssetTypeAssetTypeCreditAlphanum4, ts.Type)
	}

	_, found = MaybeGetAsset(r, "selling_")
	tt.Assert.False(found)
}

func TestActionMaybeGetAsset(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	action := makeTestAction()

	ts, found := action.MaybeGetAsset("native_")
	if tt.Assert.True(found) {
		tt.Assert.Equal(xdr.AssetTypeAssetTypeNative, ts.Type)
	}

	ts, found = action.MaybeGetAsset("4_")
	if tt.Assert.True(found) {
		tt.Assert.Equal(xdr.AssetTypeAssetTypeCreditAlphanum4, ts.Type)
	}

	_, found = action.MaybeGetAsset("selling_")
	tt.Assert.False(found)
}

func TestActionGetCursor(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()

	// now uses the ledger state
	action := makeAction("/?cursor=now", nil)
	cursor := action.GetCursor("cursor")
	if tt.Assert.NoError(action.Err) {
		expected := toid.AfterLedger(ledger.CurrentState().HistoryLatest).String()
		tt.Assert.Equal(expected, cursor)
	}

	//Last-Event-ID overrides cursor
	action = makeTestAction()
	action.R.Header.Set("Last-Event-ID", "from_header")
	cursor = action.GetCursor("cursor")
	if tt.Assert.NoError(action.Err) {
		tt.Assert.Equal("from_header", cursor)
	}
}

func TestGetCursor(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()

	// now uses the ledger state
	r := makeAction("/?cursor=now", nil).R
	cursor, err := GetCursor(r, "cursor")
	if tt.Assert.NoError(err) {
		expected := toid.AfterLedger(ledger.CurrentState().HistoryLatest).String()
		tt.Assert.Equal(expected, cursor)
	}

	//Last-Event-ID overrides cursor
	r = makeTestAction().R
	r.Header.Set("Last-Event-ID", "from_header")
	cursor, err = GetCursor(r, "cursor")
	if tt.Assert.NoError(err) {
		tt.Assert.Equal("from_header", cursor)
	}
}

func TestValidateCursorWithinHistory(t *testing.T) {
	tt := assert.New(t)
	testCases := []struct {
		cursor string
		order  string
		valid  bool
	}{
		{
			cursor: "10",
			order:  "desc",
			valid:  true,
		},
		{
			cursor: "10-1234",
			order:  "desc",
			valid:  true,
		},
		{
			cursor: "0",
			order:  "desc",
			valid:  false,
		},
		{
			cursor: "0-1234",
			order:  "desc",
			valid:  false,
		},
		{
			cursor: "10",
			order:  "asc",
			valid:  true,
		},
		{
			cursor: "10-1234",
			order:  "asc",
			valid:  true,
		},
		{
			cursor: "0",
			order:  "asc",
			valid:  true,
		},
		{
			cursor: "0-1234",
			order:  "asc",
			valid:  true,
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("cursor: %s", tc.cursor), func(t *testing.T) {
			pq, err := db2.NewPageQuery(tc.cursor, false, tc.order, 10)
			tt.NoError(err)
			err = ValidateCursorWithinHistory(pq)

			if tc.valid {
				tt.NoError(err)
			} else {
				tt.EqualError(err, "problem: before_history")
			}
		})
	}
}

func TestGetInt32(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	action := makeTestAction()

	result := action.GetInt32("blank")
	tt.Assert.NoError(action.Err)
	tt.Assert.Equal(int32(0), result)

	result = action.GetInt32("zero")
	tt.Assert.NoError(action.Err)
	tt.Assert.Equal(int32(0), result)

	result = action.GetInt32("two")
	tt.Assert.NoError(action.Err)
	tt.Assert.Equal(int32(2), result)

	result = action.GetInt32("32max")
	tt.Assert.NoError(action.Err)
	tt.Assert.EqualValues(int32(math.MaxInt32), result)

	result = action.GetInt32("32min")
	tt.Assert.NoError(action.Err)
	tt.Assert.EqualValues(int32(math.MinInt32), result)

	result = action.GetInt32("limit")
	tt.Assert.NoError(action.Err)
	tt.Assert.EqualValues(int32(2), result)

	// overflows
	action.Err = nil
	_ = action.GetInt32("64max")
	if tt.Assert.IsType(&problem.P{}, action.Err) {
		p := action.Err.(*problem.P)
		tt.Assert.Equal("bad_request", p.Type)
		tt.Assert.Equal("64max", p.Extras["invalid_field"])
	}

	action.Err = nil
	_ = action.GetInt32("64min")
	if tt.Assert.IsType(&problem.P{}, action.Err) {
		p := action.Err.(*problem.P)
		tt.Assert.Equal("bad_request", p.Type)
		tt.Assert.Equal("64min", p.Extras["invalid_field"])
	}
}

func TestGetInt64(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	action := makeTestAction()

	result := action.GetInt64("blank")
	tt.Assert.NoError(action.Err)
	tt.Assert.EqualValues(int64(0), result)

	result = action.GetInt64("zero")
	tt.Assert.NoError(action.Err)
	tt.Assert.EqualValues(int64(0), result)

	result = action.GetInt64("two")
	tt.Assert.NoError(action.Err)
	tt.Assert.Equal(int64(2), result)

	result = action.GetInt64("64max")
	tt.Assert.NoError(action.Err)
	tt.Assert.EqualValues(int64(math.MaxInt64), result)

	result = action.GetInt64("64min")
	tt.Assert.NoError(action.Err)
	tt.Assert.Equal(int64(math.MinInt64), result)
}

func TestPositiveAmount(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	r := makeTestAction().R

	result, err := GetPositiveAmount(r, "minus_one")
	tt.Assert.Error(err)
	tt.Assert.Equal(xdr.Int64(0), result)

	result, err = GetPositiveAmount(r, "zero")
	tt.Assert.Error(err)
	tt.Assert.Equal(xdr.Int64(0), result)

	result, err = GetPositiveAmount(r, "two")
	tt.Assert.NoError(err)
	tt.Assert.Equal(xdr.Int64(20000000), result)

	result, err = GetPositiveAmount(r, "twenty")
	tt.Assert.NoError(err)
	tt.Assert.Equal(xdr.Int64(200000000), result)
}

func TestActionGetLimit(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()

	// happy path
	action := makeTestAction()
	limit := action.GetLimit("limit", 5, 200)
	if tt.Assert.NoError(action.Err) {
		tt.Assert.Equal(uint64(2), limit)
	}

	action = makeAction("/?limit=200", nil)
	limit = action.GetLimit("limit", 5, 200)
	if tt.Assert.NoError(action.Err) {
		tt.Assert.Equal(uint64(200), limit)
	}

	// defaults
	action = makeAction("/", nil)
	limit = action.GetLimit("limit", 5, 200)
	if tt.Assert.NoError(action.Err) {
		tt.Assert.Equal(uint64(5), limit)
	}

	action = makeAction("/?limit=", nil)
	limit = action.GetLimit("limit", 5, 200)
	if tt.Assert.NoError(action.Err) {
		tt.Assert.Equal(uint64(5), limit)
	}

	// invalids
	action = makeAction("/?limit=0", nil)
	_ = action.GetLimit("limit", 5, 200)
	tt.Assert.Error(action.Err)

	action = makeAction("/?limit=-1", nil)
	_ = action.GetLimit("limit", 5, 200)
	tt.Assert.Error(action.Err)

	action = makeAction("/?limit=201", nil)
	_ = action.GetLimit("limit", 5, 200)
	tt.Assert.Error(action.Err)
}

func TestGetLimit(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()

	// happy path
	r := makeTestAction().R
	limit, err := GetLimit(r, "limit", 5, 200)
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(uint64(2), limit)
	}

	r = makeAction("/?limit=200", nil).R
	limit, err = GetLimit(r, "limit", 5, 200)
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(uint64(200), limit)
	}

	// defaults
	r = makeAction("/", nil).R
	limit, err = GetLimit(r, "limit", 5, 200)
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(uint64(5), limit)
	}

	r = makeAction("/?limit=", nil).R
	limit, err = GetLimit(r, "limit", 5, 200)
	if tt.Assert.NoError(err) {
		tt.Assert.Equal(uint64(5), limit)
	}

	// invalids
	r = makeAction("/?limit=0", nil).R
	_, err = GetLimit(r, "limit", 5, 200)
	tt.Assert.Error(err)

	r = makeAction("/?limit=-1", nil).R
	_, err = GetLimit(r, "limit", 5, 200)
	tt.Assert.Error(err)

	r = makeAction("/?limit=201", nil).R
	_, err = GetLimit(r, "limit", 5, 200)
	tt.Assert.Error(err)
}

func TestActionGetPageQuery(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	action := makeTestAction()

	// happy path
	pq := action.GetPageQuery()
	tt.Assert.NoError(action.Err)
	tt.Assert.Equal("123456", pq.Cursor)
	tt.Assert.Equal(uint64(2), pq.Limit)
	tt.Assert.Equal("asc", pq.Order)

	// regression: GetPagQuery does not overwrite err
	action = makeAction("/?limit=foo", nil)
	_ = action.GetLimit("limit", 1, 200)
	tt.Assert.Error(action.Err)
	_ = action.GetPageQuery()
	tt.Assert.Error(action.Err)

	// regression: https://github.com/stellar/go/services/horizon/internal/issues/372
	// (limit of 0 turns into 10)
	makeAction("/?limit=0", nil)
	_ = action.GetPageQuery()
	tt.Assert.Error(action.Err)
}

func TestGetPageQuery(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	r := makeTestAction().R

	// happy path
	pq, err := GetPageQuery(r)
	tt.Assert.NoError(err)
	tt.Assert.Equal("123456", pq.Cursor)
	tt.Assert.Equal(uint64(2), pq.Limit)
	tt.Assert.Equal("asc", pq.Order)

	// regression: GetPagQuery does not overwrite err
	r = makeAction("/?limit=foo", nil).R
	_, err = GetLimit(r, "limit", 1, 200)
	tt.Assert.Error(err)
	_, err = GetPageQuery(r)
	tt.Assert.Error(err)

	// regression: https://github.com/stellar/go/services/horizon/internal/issues/372
	// (limit of 0 turns into 10)
	r = makeAction("/?limit=0", nil).R
	_, err = GetPageQuery(r)
	tt.Assert.Error(err)
}

func TestGetString(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	action := makeTestAction()

	tt.Assert.Equal("123456", action.GetString("cursor"))
	action.R.Form = url.Values{
		"cursor": {"goodbye"},
	}
	tt.Assert.Equal("goodbye", action.GetString("cursor"))
}

func TestPath(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	action := makeTestAction()

	tt.Assert.Equal("/foo-bar/blah", action.Path())
}

func TestBaseGetURLParam(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	action := makeTestAction()

	val, ok := action.GetURLParam("two")
	tt.Assert.Equal("2", val)
	tt.Assert.Equal(true, ok)

	// valid empty string
	val, ok = action.GetURLParam("blank")
	tt.Assert.Equal("", val)
	tt.Assert.Equal(true, ok)

	// url param not found
	val, ok = action.GetURLParam("foobarcursor")
	tt.Assert.Equal("", val)
	tt.Assert.Equal(false, ok)
}

func TestGetURLParam(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	r := makeAction("/accounts/{account_id}/operations?limit=100", nil).R

	// simulates a request where the named param is not passed.
	// Regression for https://github.com/stellar/go/issues/1965
	rctx := chi.RouteContext(r.Context())
	rctx.URLParams.Keys = []string{
		"account_id",
	}

	val, ok := GetURLParam(r, "account_id")
	tt.Assert.Empty(val)
	tt.Assert.False(ok)
}

func TestGetAssets(t *testing.T) {
	rctx := chi.NewRouteContext()

	path := "/foo-bar/blah?assets="
	for _, testCase := range []struct {
		name           string
		value          string
		expectedAssets []xdr.Asset
		expectedError  string
	}{
		{
			"empty list",
			"",
			[]xdr.Asset{},
			"",
		},
		{
			"native",
			"native",
			[]xdr.Asset{xdr.MustNewNativeAsset()},
			"",
		},
		{
			"asset does not contain :",
			"invalid-asset",
			[]xdr.Asset{},
			"invalid-asset is not a valid asset",
		},
		{
			"asset contains more than one :",
			"usd:GAEDTJ4PPEFVW5XV2S7LUXBEHNQMX5Q2GM562RJGOQG7GVCE5H3HIB4V:",
			[]xdr.Asset{},
			"is not a valid asset",
		},
		{
			"unicode asset code",
			"üsd:GAEDTJ4PPEFVW5XV2S7LUXBEHNQMX5Q2GM562RJGOQG7GVCE5H3HIB4V",
			[]xdr.Asset{},
			"contains an invalid asset code",
		},
		{
			"asset code must be alpha numeric",
			"!usd:GAEDTJ4PPEFVW5XV2S7LUXBEHNQMX5Q2GM562RJGOQG7GVCE5H3HIB4V",
			[]xdr.Asset{},
			"contains an invalid asset code",
		},
		{
			"asset code contains backslash",
			"usd\\x23:GAEDTJ4PPEFVW5XV2S7LUXBEHNQMX5Q2GM562RJGOQG7GVCE5H3HIB4V",
			[]xdr.Asset{},
			"contains an invalid asset code",
		},
		{
			"contains null characters",
			"abcde\\x00:GAEDTJ4PPEFVW5XV2S7LUXBEHNQMX5Q2GM562RJGOQG7GVCE5H3HIB4V",
			[]xdr.Asset{},
			"contains an invalid asset code",
		},
		{
			"asset code is too short",
			":GAEDTJ4PPEFVW5XV2S7LUXBEHNQMX5Q2GM562RJGOQG7GVCE5H3HIB4V",
			[]xdr.Asset{},
			"is not a valid asset",
		},
		{
			"asset code is too long",
			"0123456789abc:GAEDTJ4PPEFVW5XV2S7LUXBEHNQMX5Q2GM562RJGOQG7GVCE5H3HIB4V",
			[]xdr.Asset{},
			"is not a valid asset",
		},
		{
			"issuer is empty",
			"usd:",
			[]xdr.Asset{},
			"contains an invalid issuer",
		},
		{
			"issuer is invalid",
			"usd:kkj9808;l",
			[]xdr.Asset{},
			"contains an invalid issuer",
		},
		{
			"validation succeeds",
			"usd:GAEDTJ4PPEFVW5XV2S7LUXBEHNQMX5Q2GM562RJGOQG7GVCE5H3HIB4V,usdabc:GAEDTJ4PPEFVW5XV2S7LUXBEHNQMX5Q2GM562RJGOQG7GVCE5H3HIB4V",
			[]xdr.Asset{
				xdr.MustNewCreditAsset("usd", "GAEDTJ4PPEFVW5XV2S7LUXBEHNQMX5Q2GM562RJGOQG7GVCE5H3HIB4V"),
				xdr.MustNewCreditAsset("usdabc", "GAEDTJ4PPEFVW5XV2S7LUXBEHNQMX5Q2GM562RJGOQG7GVCE5H3HIB4V"),
			},
			"",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			tt := assert.New(t)
			r, err := http.NewRequest("GET", path+url.QueryEscape(testCase.value), nil)
			tt.NoError(err)

			ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
			r = r.WithContext(context.WithValue(ctx, &horizonContext.RequestContextKey, r))

			assets, err := GetAssets(r, "assets")
			if testCase.expectedError == "" {
				tt.NoError(err)
				tt.Len(assets, len(testCase.expectedAssets))
				for i := range assets {
					tt.Equal(testCase.expectedAssets[i], assets[i])
				}
			} else {
				p := err.(*problem.P)
				tt.Equal(p.Extras["invalid_field"], "assets")
				tt.Contains(p.Extras["reason"], testCase.expectedError)
			}
		})
	}
}

func TestFullURL(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	action := makeTestAction()

	url := FullURL(action.R.Context())
	tt.Assert.Equal("http:///foo-bar/blah?limit=2&cursor=123456", url.String())
}

func TestGetParams(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()

	type QueryParams struct {
		SellingBuyingAssetQueryParams `valid:"-"`
		Account                       string `schema:"account_id" valid:"accountID"`
	}

	account := "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H"

	// Simulate chi's URL params. The following would be equivalent to having a
	// chi route like the following `/accounts/{account_id}`
	urlParams := map[string]string{
		"account_id":           account,
		"selling_asset_type":   "credit_alphanum4",
		"selling_asset_code":   "USD",
		"selling_asset_issuer": account,
	}

	r := makeAction("/transactions?limit=2&cursor=123456&order=desc", urlParams).R
	qp := QueryParams{}
	err := GetParams(&qp, r)

	tt.Assert.NoError(err)
	tt.Assert.Equal(account, qp.Account)
	selling, err := qp.Selling()
	tt.Assert.NoError(err)
	tt.Assert.NotNil(selling)
	tt.Assert.True(usd.Equals(*selling))

	urlParams = map[string]string{
		"account_id":         account,
		"selling_asset_type": "native",
	}

	r = makeAction("/transactions?limit=2&cursor=123456&order=desc", urlParams).R
	qp = QueryParams{}
	err = GetParams(&qp, r)

	tt.Assert.NoError(err)
	native := xdr.MustNewNativeAsset()
	selling, err = qp.Selling()
	tt.Assert.NoError(err)
	tt.Assert.NotNil(selling)
	tt.Assert.True(native.Equals(*selling))

	urlParams = map[string]string{"account_id": "1"}
	r = makeAction("/transactions?limit=2&cursor=123456&order=desc", urlParams).R
	qp = QueryParams{}
	err = GetParams(&qp, r)

	if tt.Assert.IsType(&problem.P{}, err) {
		p := err.(*problem.P)
		tt.Assert.Equal("bad_request", p.Type)
		tt.Assert.Equal("account_id", p.Extras["invalid_field"])
		tt.Assert.Equal(
			"Account ID must start with `G` and contain 56 alphanum characters",
			p.Extras["reason"],
		)
	}

	urlParams = map[string]string{
		"account_id": account,
	}
	r = makeAction(fmt.Sprintf("/transactions?account_id=%s", account), urlParams).R
	err = GetParams(&qp, r)

	tt.Assert.Error(err)
	if tt.Assert.IsType(&problem.P{}, err) {
		p := err.(*problem.P)
		tt.Assert.Equal("bad_request", p.Type)
		tt.Assert.Equal("account_id", p.Extras["invalid_field"])
		tt.Assert.Equal(
			"The parameter should not be included in the request",
			p.Extras["reason"],
		)
	}
}

type ParamsValidator struct {
	Account string `schema:"account_id" valid:"required"`
}

func (pv ParamsValidator) Validate() error {
	return problem.MakeInvalidFieldProblem(
		"Name",
		errors.New("Invalid"),
	)
}

func TestGetParamsCustomValidator(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()

	urlParams := map[string]string{"account_id": "1"}
	r := makeAction("/transactions", urlParams).R
	qp := ParamsValidator{}
	err := GetParams(&qp, r)

	if tt.Assert.IsType(&problem.P{}, err) {
		p := err.(*problem.P)
		tt.Assert.Equal("bad_request", p.Type)
		tt.Assert.Equal("Name", p.Extras["invalid_field"])
	}
}

func makeTestAction() *Base {
	return makeAction("/foo-bar/blah?limit=2&cursor=123456", testURLParams())
}

func makeAction(path string, body map[string]string) *Base {
	rctx := chi.NewRouteContext()
	for k, v := range body {
		rctx.URLParams.Add(k, v)
	}

	r, _ := http.NewRequest("GET", path, nil)

	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	r = r.WithContext(context.WithValue(ctx, &horizonContext.RequestContextKey, r))
	action := &Base{
		R: r,
	}
	return action
}

func testURLParams() map[string]string {
	return map[string]string{
		"blank":                   "",
		"minus_one":               "-1",
		"zero":                    "0",
		"two":                     "2",
		"twenty":                  "20",
		"32min":                   fmt.Sprint(math.MinInt32),
		"32max":                   fmt.Sprint(math.MaxInt32),
		"64min":                   fmt.Sprint(math.MinInt64),
		"64max":                   fmt.Sprint(math.MaxInt64),
		"native_asset_type":       "native",
		"4_asset_type":            "credit_alphanum4",
		"4_asset_code":            "USD",
		"4_asset_issuer":          "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		"12_asset_type":           "credit_alphanum12",
		"12_asset_code":           "USD",
		"12_asset_issuer":         "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		"long_4_asset_type":       "credit_alphanum4",
		"long_4_asset_code":       "SPOOON",
		"long_4_asset_issuer":     "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		"long_12_asset_type":      "credit_alphanum12",
		"long_12_asset_code":      "OHMYGODITSSOLONG",
		"long_12_asset_issuer":    "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
		"valid_tx_id":             "aa168f12124b7c196c0adaee7c73a64d37f99428cacb59a91ff389626845e7cf",
		"invalid_uppercase_tx_id": "AA168F12124B7C196C0ADAEE7C73A64D37F99428CACB59A91FF389626845E7CF",
		"invalid_too_short_tx_id": "aa168f12124b7c196c0adaee7c73a64d37f99428cacb59a91ff389626845e7",
	}
}

func makeRequest(
	t *testing.T,
	queryParams map[string]string,
	routeParams map[string]string,
	session *db.Session,
) *http.Request {
	request, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	query := url.Values{}
	for key, value := range queryParams {
		query.Set(key, value)
	}
	request.URL.RawQuery = query.Encode()

	chiRouteContext := chi.NewRouteContext()
	for key, value := range routeParams {
		chiRouteContext.URLParams.Add(key, value)
	}
	ctx := context.WithValue(
		context.WithValue(context.Background(), chi.RouteCtxKey, chiRouteContext),
		&horizonContext.SessionContextKey,
		session,
	)

	return request.WithContext(ctx)
}

func TestGetURIParams(t *testing.T) {
	tt := assert.New(t)
	type QueryParams struct {
		SellingBuyingAssetQueryParams `valid:"-"`
		Account                       string `schema:"account_id" valid:"accountID"`
	}

	expected := []string{
		"selling_asset_type",
		"selling_asset_issuer",
		"selling_asset_code",
		"buying_asset_type",
		"buying_asset_issuer",
		"buying_asset_code",
		"selling",
		"buying",
		"account_id",
	}

	qp := QueryParams{}
	tt.Equal(expected, GetURIParams(&qp, false))
}
