package integration

import (
	"context"
	"fmt"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/ingest/ledgerbackend"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon/effects"
	"github.com/stellar/go/services/horizon/internal/test/integration"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

func TestMuxedOperations(t *testing.T) {
	itest := integration.NewTest(t, integration.Config{})

	sponsored := keypair.MustRandom()
	// Is there an easier way?
	sponsoredMuxed := xdr.MustMuxedAddress(sponsored.Address())
	sponsoredMuxed.Type = xdr.CryptoKeyTypeKeyTypeMuxedEd25519
	sponsoredMuxed.Med25519 = &xdr.MuxedAccountMed25519{
		Ed25519: *sponsoredMuxed.Ed25519,
		Id:      100,
	}

	master := itest.Master()
	masterMuxed := xdr.MustMuxedAddress(master.Address())
	masterMuxed.Type = xdr.CryptoKeyTypeKeyTypeMuxedEd25519
	masterMuxed.Med25519 = &xdr.MuxedAccountMed25519{
		Ed25519: *masterMuxed.Ed25519,
		Id:      200,
	}

	ops := []txnbuild.Operation{
		&txnbuild.BeginSponsoringFutureReserves{
			SponsoredID: sponsored.Address(),
		},
		&txnbuild.CreateAccount{
			Destination: sponsored.Address(),
			Amount:      "100",
		},
		&txnbuild.ChangeTrust{
			SourceAccount: sponsoredMuxed.Address(),
			Line:          txnbuild.CreditAsset{Code: "ABCD", Issuer: master.Address()}.MustToChangeTrustAsset(),
			Limit:         txnbuild.MaxTrustlineLimit,
		},
		&txnbuild.ManageSellOffer{
			SourceAccount: sponsoredMuxed.Address(),
			Selling:       txnbuild.NativeAsset{},
			Buying:        txnbuild.CreditAsset{Code: "ABCD", Issuer: master.Address()},
			Amount:        "3",
			Price:         xdr.Price{N: 1, D: 1},
		},
		// This will generate a trade effect:
		&txnbuild.ManageSellOffer{
			SourceAccount: masterMuxed.Address(),
			Selling:       txnbuild.CreditAsset{Code: "ABCD", Issuer: master.Address()},
			Buying:        txnbuild.NativeAsset{},
			Amount:        "3",
			Price:         xdr.Price{N: 1, D: 1},
		},
		&txnbuild.ManageData{
			SourceAccount: sponsoredMuxed.Address(),
			Name:          "test",
			Value:         []byte("test"),
		},
		&txnbuild.Payment{
			SourceAccount: sponsoredMuxed.Address(),
			Destination:   master.Address(),
			Amount:        "1",
			Asset:         txnbuild.NativeAsset{},
		},
		&txnbuild.CreateClaimableBalance{
			SourceAccount: sponsoredMuxed.Address(),
			Amount:        "2",
			Asset:         txnbuild.NativeAsset{},
			Destinations: []txnbuild.Claimant{
				txnbuild.NewClaimant(keypair.MustRandom().Address(), nil),
			},
		},
		&txnbuild.EndSponsoringFutureReserves{
			SourceAccount: sponsored.Address(),
		},
	}
	txResp, err := itest.SubmitMultiSigOperations(itest.MasterAccount(), []*keypair.Full{master, sponsored}, ops...)
	assert.NoError(t, err)
	assert.True(t, txResp.Successful)

	ops = []txnbuild.Operation{
		// Remove subentries to be able to merge account
		&txnbuild.Payment{
			SourceAccount: sponsoredMuxed.Address(),
			Destination:   master.Address(),
			Amount:        "3",
			Asset:         txnbuild.CreditAsset{Code: "ABCD", Issuer: master.Address()},
		},
		&txnbuild.ChangeTrust{
			SourceAccount: sponsoredMuxed.Address(),
			Line:          txnbuild.CreditAsset{Code: "ABCD", Issuer: master.Address()}.MustToChangeTrustAsset(),
			Limit:         "0",
		},
		&txnbuild.ManageData{
			SourceAccount: sponsoredMuxed.Address(),
			Name:          "test",
		},
		&txnbuild.AccountMerge{
			SourceAccount: sponsoredMuxed.Address(),
			Destination:   masterMuxed.Address(),
		},
	}
	txResp, err = itest.SubmitMultiSigOperations(itest.MasterAccount(), []*keypair.Full{master, sponsored}, ops...)
	assert.NoError(t, err)
	assert.True(t, txResp.Successful)

	// Check if no 5xx after processing the tx above
	// TODO expand it to test actual muxed fields
	_, err = itest.Client().Operations(horizonclient.OperationRequest{Limit: 200})
	assert.NoError(t, err, "/operations failed")

	_, err = itest.Client().Payments(horizonclient.OperationRequest{Limit: 200})
	assert.NoError(t, err, "/payments failed")

	effectsPage, err := itest.Client().Effects(horizonclient.EffectRequest{Limit: 200})
	assert.NoError(t, err, "/effects failed")

	for _, effect := range effectsPage.Embedded.Records {
		if effect.GetType() == "trade" {
			trade := effect.(effects.Trade)
			oneSet := trade.AccountMuxedID != 0 || trade.SellerMuxedID != 0
			assert.True(t, oneSet, "at least one of account_muxed_id, seller_muxed_id must be set")
		}
	}

	time.Sleep(time.Second * 5)

	captiveCoreConfig := ledgerbackend.CaptiveCoreConfig{}
	captiveCoreConfig.BinaryPath = os.Getenv("HORIZON_INTEGRATION_TESTS_CAPTIVE_CORE_BIN")
	captiveCoreConfig.HistoryArchiveURLs = []string{itest.GetDefaultArgs()["history-archive-urls"]}
	captiveCoreConfig.NetworkPassphrase = integration.StandaloneNetworkPassphrase
	captiveCoreConfig.CheckpointFrequency = 8
	confName, _, cleanup := CreateCaptiveCoreConfig(SimpleCaptiveCoreToml)
	kk := ledgerbackend.CaptiveCoreTomlParams{
		NetworkPassphrase:  captiveCoreConfig.NetworkPassphrase,
		HistoryArchiveURLs: captiveCoreConfig.HistoryArchiveURLs,
	}

	captiveCoreToml, _ := ledgerbackend.NewCaptiveCoreTomlFromFile(confName, kk)
	captiveCoreConfig.Toml = captiveCoreToml
	defer cleanup()

	var captiveCore *ledgerbackend.CaptiveStellarCore
	captiveCore, err = ledgerbackend.NewCaptive(captiveCoreConfig)
	if err != nil {
		t.Fatal(err)
	}
	cc := context.Background()
	err = captiveCore.PrepareRange(cc, ledgerbackend.BoundedRange(uint32(txResp.Ledger), uint32(txResp.Ledger)))
	defer captiveCore.Close()

	if err != nil {
		t.Fatal(err)
	}

	ll, _ := captiveCore.GetLedger(cc, uint32(txResp.Ledger))

	var successfulTransactions, failedTransactions int
	var operationsInSuccessful, operationsInFailed int

	txReader, _ := ingest.NewLedgerTransactionReaderFromLedgerCloseMeta(
		captiveCoreConfig.NetworkPassphrase, ll,
	)
	//panicIf(err)
	defer txReader.Close()

	// Read each transaction within the ledger, extract its operations, and
	// accumulate the statistics we're interested in.
	for {
		ltx, err := txReader.Read()
		if err == io.EOF {
			break
		}
		//panicIf(err)

		envelope := ltx.Envelope
		operationCount := len(envelope.Operations())
		if ltx.Result.Successful() {
			successfulTransactions++
			operationsInSuccessful += operationCount
		} else {
			failedTransactions++
			operationsInFailed += operationCount
		}
	}

	fmt.Println("\nDone. Results:")
	fmt.Printf("  - total transactions: %d\n", successfulTransactions+failedTransactions)
	fmt.Printf("  - succeeded / failed: %d / %d\n", successfulTransactions, failedTransactions)
	fmt.Printf("  - total operations:   %d\n", operationsInSuccessful+operationsInFailed)
	fmt.Printf("  - succeeded / failed: %d / %d\n", operationsInSuccessful, operationsInFailed)

	t.Logf("----------- This is %v, %v", ll.LedgerSequence(), ll.TransactionHash(0))

}
