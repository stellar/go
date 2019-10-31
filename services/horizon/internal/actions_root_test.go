package horizon

import (
	"encoding/json"
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
	}
}

func TestRootActionWithIngestion(t *testing.T) {
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
	ht.App.config.EnableExperimentalIngestion = true

	w := ht.Get("/")

	if ht.Assert.Equal(200, w.Code) {
		var actual horizon.Root
		err := json.Unmarshal(w.Body.Bytes(), &actual)
		ht.Require.NoError(err)
		ht.Assert.Equal(
			"http://localhost/accounts{?signer,asset,cursor,limit,order}",
			actual.Links.Accounts.Href,
		)
		ht.Assert.Equal(
			"http://localhost/offers{?selling_asset_type,selling_asset_issuer,selling_asset_code,buying_asset_type,buying_asset_issuer,buying_asset_code,seller,cursor,limit,order}",
			actual.Links.Offers.Href,
		)

	}
}
