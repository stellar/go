package integration

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/services/horizon/internal/txnbuild"
	"github.com/stretchr/testify/assert"
)

func TestProtocol14StateVerifier(t *testing.T) {
	itest := test.NewIntegrationTest(t, protocol14Config)
	defer itest.Close()

	sponsored := keypair.MustRandom()
	sponsoredSource := &txnbuild.SimpleAccount{
		AccountID: sponsored.Address(),
		Sequence:  1,
	}
	signer := keypair.MustRandom()

	// The operations below create a sponsorship sandwich, sponsoring an
	// account, its trustlines, offers, data, and claimable balances.
	// TODO multiple signers and a sponsor at non-first position
	master := itest.Master()
	ops := []txnbuild.Operation{
		&txnbuild.BeginSponsoringFutureReserves{
			SponsoredID: sponsored.Address(),
		},
		&txnbuild.CreateAccount{
			Destination: sponsored.Address(),
			Amount:      "100",
		},
		&txnbuild.CreateAccount{
			Destination: signer.Address(),
			Amount:      "100",
		},
		&txnbuild.ChangeTrust{
			SourceAccount: sponsoredSource,
			Line:          txnbuild.CreditAsset{"ABCD", master.Address()},
			Limit:         txnbuild.MaxTrustlineLimit,
		},
		&txnbuild.ManageSellOffer{
			SourceAccount: sponsoredSource,
			Selling:       txnbuild.NativeAsset{},
			Buying:        txnbuild.CreditAsset{"ABCD", master.Address()},
			Amount:        "3",
			Price:         "1",
		},
		&txnbuild.ManageData{
			SourceAccount: sponsoredSource,
			Name:          "test",
			Value:         []byte("test"),
		},
		&txnbuild.CreateClaimableBalance{
			SourceAccount: sponsoredSource,
			Amount:        "2",
			Asset:         txnbuild.NativeAsset{},
			Destinations: []txnbuild.Claimant{
				txnbuild.NewClaimant(keypair.MustRandom().Address(), nil),
			},
		},
		&txnbuild.SetOptions{
			SourceAccount: sponsoredSource,
			Signer: &txnbuild.Signer{
				Address: signer.Address(),
				Weight:  3,
			},
		},
		&txnbuild.EndSponsoringFutureReserves{
			SourceAccount: sponsoredSource,
		},
	}
	txResp, err := itest.SubmitMultiSigOperations(itest.MasterAccount(), []*keypair.Full{master, sponsored}, ops...)
	assert.NoError(t, err)
	assert.True(t, txResp.Successful)

	// Wait for the first checkpoint ledger
	for !itest.LedgerIngested(63) {
		t.Log("First checkpoint ledger (63) not closed yet...")
		time.Sleep(5 * time.Second)
	}

	var metrics string

	// Check metrics until state verification run
	for i := 0; i < 60; i++ {
		t.Logf("Checking metrics (%d attempt)\n", i)
		res, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", itest.AdminPort()))
		assert.NoError(t, err)

		metricsBytes, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		assert.NoError(t, err)
		metrics = string(metricsBytes)

		stateInvalid := strings.Contains(metrics, "horizon_ingest_state_invalid 1")
		assert.False(t, stateInvalid, "State is invalid!")

		notVerifiedYet := strings.Contains(metrics, "horizon_ingest_state_verify_duration_seconds_count 0")
		if notVerifiedYet {
			time.Sleep(time.Second)
			continue
		}

		return
	}

	t.Fatal("State verification not run...")
}
