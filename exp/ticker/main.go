package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stellar/go/exp/ticker/internal/tickerdb"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func writeResultsToFile(jsonBytes []byte, filename string) (numBytes int, err error) {
	path := filepath.Join(".", "tmp")
	_ = os.Mkdir(path, os.ModePerm) // ignore if dir already exists

	f, err := os.Create(filepath.Join(".", "tmp", filename))
	check(err)
	defer f.Close()

	numBytes, err = f.Write(jsonBytes)
	fmt.Printf("Wrote %d bytes to %s\n", numBytes, filename)
	f.Sync()

	return
}

func main() {
	// Temporary main function to run / test packages
	// c := horizonclient.DefaultPublicNetClient
	// tomlAssetList, err := scraper.FetchAllAssets(c, 0, 500)
	// check(err)

	// jsonAssets, err := json.MarshalIndent(tomlAssetList, "", "\t")
	// check(err)

	// writeResultsToFile(jsonAssets, "assets.json")
	dbconn, err := sqlx.Connect("postgres", "user=alexandre dbname=stellarticker01 sslmode=disable")
	check(err)

	var session tickerdb.TickerSession
	session.DB = dbconn
	asset := tickerdb.Asset{
		Code:                    "BTC",
		Issuer:                  "Alexandre Cordeiro",
		Type:                    "crypto",
		NumAccounts:             439,
		AuthRequired:            true,
		AuthRevocable:           false,
		Amount:                  1098312903.0931232901,
		AssetControlledByDomain: true,
		IsValid:                 true,
		LastValid:               time.Now(),
		LastChecked:             time.Now(),
	}

	err = session.InsertAsset(&asset)
	check(err)
}
