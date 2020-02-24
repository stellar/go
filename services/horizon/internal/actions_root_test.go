package horizon

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/test"
)

func TestRootAction(t *testing.T) {
	ht := StartHTTPTest(t, "base")
	defer ht.Finish()

	server := test.NewStaticMockServer(`{
			"info": {
				"network": "test",
				"build": "test-core",
				"ledger": {
					"version": 3
				},
				"protocol_version": 4
			}
		}`)
	defer server.Close()

	ht.App.horizonVersion = "test-horizon"
	ht.App.config.StellarCoreURL = server.URL
	ht.App.config.NetworkPassphrase = "test"
	ht.App.UpdateStellarCoreInfo()

	w := ht.Get("/")

	if ht.Assert.Equal(200, w.Code) {
		var actual horizon.Root
		err := json.Unmarshal(w.Body.Bytes(), &actual)
		ht.Require.NoError(err)
		ht.Assert.Equal("test-horizon", actual.HorizonVersion)
		ht.Assert.Equal("test-core", actual.StellarCoreVersion)
		ht.Assert.Equal(int32(4), actual.CoreSupportedProtocolVersion)
		ht.Assert.Equal(int32(3), actual.CurrentProtocolVersion)

		err = json.Unmarshal(w.Body.Bytes(), &actual)
		ht.Require.NoError(err)
		ht.Assert.Equal(
			"http://localhost/accounts{?signer,asset,cursor,limit,order}",
			actual.Links.Accounts.Href,
		)
		ht.Assert.Equal(
			"http://localhost/offers{?selling,buying,seller,cursor,limit,order}",
			actual.Links.Offers.Href,
		)

		params := []string{
			"destination_account",
			"destination_assets",
			"source_asset_type",
			"source_asset_issuer",
			"source_asset_code",
			"source_amount",
		}

		ht.Assert.Equal(
			"http://localhost/paths/strict-send{?"+strings.Join(params, ",")+"}",
			actual.Links.StrictSendPaths.Href,
		)

		params = []string{
			"source_assets",
			"source_account",
			"destination_account",
			"destination_asset_type",
			"destination_asset_issuer",
			"destination_asset_code",
			"destination_amount",
		}

		ht.Assert.Equal(
			"http://localhost/paths/strict-receive{?"+strings.Join(params, ",")+"}",
			actual.Links.StrictReceivePaths.Href,
		)
	}
}
