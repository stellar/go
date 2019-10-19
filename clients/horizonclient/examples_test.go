package horizonclient_test

import (
	"fmt"

	"github.com/stellar/go/clients/horizonclient"
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
