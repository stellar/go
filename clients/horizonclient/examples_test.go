package horizonclient_test

import (
	"context"
	"fmt"
	"time"

	"github.com/stellar/go/clients/horizonclient"
	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/effects"
)

func ExampleClient_NextAssetsPage() {
	client := horizonclient.DefaultPublicNetClient
	// assets for asset issuer
	assetRequest := horizonclient.AssetRequest{ForAssetIssuer: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU",
		Limit: 20}
	asset, err := client.Assets(assetRequest)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(asset)

	// all assets
	assetRequest = horizonclient.AssetRequest{}
	asset, err = client.Assets(assetRequest)
	if err != nil {
		fmt.Println(err)
		return
	}

	// next page
	nextPage, err := client.NextAssetsPage(asset)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(nextPage)
}

func ExampleClient_NextEffectsPage() {
	client := horizonclient.DefaultPublicNetClient
	// all effects
	effectRequest := horizonclient.EffectRequest{Limit: 20}
	efp, err := client.Effects(effectRequest)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(efp)

	// get next pages.
	recordsFound := false
	if len(efp.Embedded.Records) > 0 {
		recordsFound = true
	}
	page := efp
	// get the next page of records if recordsFound is true
	for recordsFound {
		// next page
		nextPage, err := client.NextEffectsPage(page)
		if err != nil {
			fmt.Println(err)
			return
		}

		page = nextPage
		if len(nextPage.Embedded.Records) == 0 {
			recordsFound = false
		}
		fmt.Println(nextPage)
	}
}

func ExampleClient_NextLedgersPage() {
	client := horizonclient.DefaultPublicNetClient
	// all ledgers
	ledgerRequest := horizonclient.LedgerRequest{Limit: 20}
	ledgers, err := client.Ledgers(ledgerRequest)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(ledgers)

	// get next pages.
	recordsFound := false
	if len(ledgers.Embedded.Records) > 0 {
		recordsFound = true
	}
	page := ledgers
	// get the next page of records if recordsFound is true
	for recordsFound {
		// next page
		nextPage, err := client.NextLedgersPage(page)
		if err != nil {
			fmt.Println(err)
			return
		}

		page = nextPage
		if len(nextPage.Embedded.Records) == 0 {
			recordsFound = false
		}
		fmt.Println(nextPage)
	}
}

func ExampleClient_NextOffersPage() {
	client := horizonclient.DefaultPublicNetClient
	// all offers
	offerRequest := horizonclient.OfferRequest{ForAccount: "GAQHWQYBBW272OOXNQMMLCA5WY2XAZPODGB7Q3S5OKKIXVESKO55ZQ7C", Limit: 20}
	offers, err := client.Offers(offerRequest)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(offers)

	// get next pages.
	recordsFound := false
	if len(offers.Embedded.Records) > 0 {
		recordsFound = true
	}
	page := offers
	// get the next page of records if recordsFound is true
	for recordsFound {
		// next page
		nextPage, err := client.NextOffersPage(page)
		if err != nil {
			fmt.Println(err)
			return
		}

		page = nextPage
		if len(nextPage.Embedded.Records) == 0 {
			recordsFound = false
		}
		fmt.Println(nextPage)
	}
}

func ExampleClient_PrevAssetsPage() {
	client := horizonclient.DefaultPublicNetClient
	// assets for asset issuer
	assetRequest := horizonclient.AssetRequest{ForAssetIssuer: "GCLWGQPMKXQSPF776IU33AH4PZNOOWNAWGGKVTBQMIC5IMKUNP3E6NVU",
		Limit: 20}
	asset, err := client.Assets(assetRequest)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(asset)

	// all assets
	assetRequest = horizonclient.AssetRequest{}
	asset, err = client.Assets(assetRequest)
	if err != nil {
		fmt.Println(err)
		return
	}

	// next page
	prevPage, err := client.PrevAssetsPage(asset)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(prevPage)
}

func ExampleClient_PrevEffectsPage() {
	client := horizonclient.DefaultPublicNetClient
	// all effects
	effectRequest := horizonclient.EffectRequest{Limit: 20}
	efp, err := client.Effects(effectRequest)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(efp)

	// get prev pages.
	recordsFound := false
	if len(efp.Embedded.Records) > 0 {
		recordsFound = true
	}
	page := efp
	// get the prev page of records if recordsFound is true
	for recordsFound {
		// prev page
		prevPage, err := client.PrevEffectsPage(page)
		if err != nil {
			fmt.Println(err)
			return
		}

		page = prevPage
		if len(prevPage.Embedded.Records) == 0 {
			recordsFound = false
		}
		fmt.Println(prevPage)
	}
}

func ExampleClient_PrevLedgersPage() {
	client := horizonclient.DefaultPublicNetClient
	// all ledgers
	ledgerRequest := horizonclient.LedgerRequest{Limit: 20}
	ledgers, err := client.Ledgers(ledgerRequest)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(ledgers)

	// get prev pages.
	recordsFound := false
	if len(ledgers.Embedded.Records) > 0 {
		recordsFound = true
	}
	page := ledgers
	// get the prev page of records if recordsFound is true
	for recordsFound {
		// prev page
		prevPage, err := client.PrevLedgersPage(page)
		if err != nil {
			fmt.Println(err)
			return
		}

		page = prevPage
		if len(prevPage.Embedded.Records) == 0 {
			recordsFound = false
		}
		fmt.Println(prevPage)
	}
}

func ExampleClient_PrevOffersPage() {
	client := horizonclient.DefaultPublicNetClient
	// all offers
	offerRequest := horizonclient.OfferRequest{ForAccount: "GAQHWQYBBW272OOXNQMMLCA5WY2XAZPODGB7Q3S5OKKIXVESKO55ZQ7C", Limit: 20}
	offers, err := client.Offers(offerRequest)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(offers)

	// get prev pages.
	recordsFound := false
	if len(offers.Embedded.Records) > 0 {
		recordsFound = true
	}
	page := offers
	// get the prev page of records if recordsFound is true
	for recordsFound {
		// prev page
		prevPage, err := client.PrevOffersPage(page)
		if err != nil {
			fmt.Println(err)
			return
		}

		page = prevPage
		if len(prevPage.Embedded.Records) == 0 {
			recordsFound = false
		}
		fmt.Println(prevPage)
	}
}

func ExampleClient_StreamEffects() {
	client := horizonclient.DefaultTestNetClient
	// all effects
	effectRequest := horizonclient.EffectRequest{Cursor: "760209215489"}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Stop streaming after 60 seconds.
		time.Sleep(60 * time.Second)
		cancel()
	}()

	printHandler := func(e effects.Effect) {
		fmt.Println(e)
	}
	err := client.StreamEffects(ctx, effectRequest, printHandler)
	if err != nil {
		fmt.Println(err)
	}
}

func ExampleClient_StreamLedgers() {
	client := horizonclient.DefaultTestNetClient
	// all ledgers from now
	ledgerRequest := horizonclient.LedgerRequest{}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Stop streaming after 60 seconds.
		time.Sleep(60 * time.Second)
		cancel()
	}()

	printHandler := func(ledger hProtocol.Ledger) {
		fmt.Println(ledger)
	}
	err := client.StreamLedgers(ctx, ledgerRequest, printHandler)
	if err != nil {
		fmt.Println(err)
	}
}

func ExampleClient_StreamOffers() {
	client := horizonclient.DefaultTestNetClient
	// offers for account
	offerRequest := horizonclient.OfferRequest{ForAccount: "GAQHWQYBBW272OOXNQMMLCA5WY2XAZPODGB7Q3S5OKKIXVESKO55ZQ7C", Cursor: "1"}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Stop streaming after 60 seconds.
		time.Sleep(60 * time.Second)
		cancel()
	}()

	printHandler := func(offer hProtocol.Offer) {
		fmt.Println(offer)
	}
	err := client.StreamOffers(ctx, offerRequest, printHandler)
	if err != nil {
		fmt.Println(err)
	}
}
