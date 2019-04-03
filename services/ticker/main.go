package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	horizonclient "github.com/stellar/go/exp/clients/horizon"
	"github.com/stellar/go/services/ticker/internal/scraper"
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
	c := horizonclient.DefaultPublicNetClient
	assetStatList, err := scraper.FetchAllAssets(c)
	check(err)

	jsonAssets, err := json.MarshalIndent(assetStatList, "", "\t")
	check(err)

	writeResultsToFile(jsonAssets, "assets.json")
}
