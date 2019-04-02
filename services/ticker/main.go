package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/stellar/go/services/ticker/internal/assetscraper"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	assetStatList, err := assetscraper.FetchAllAssets()
	check(err)

	marshAssets, err := json.MarshalIndent(assetStatList, "", "\t")
	check(err)

	path := filepath.Join(".", "tmp")
	_ = os.Mkdir(path, os.ModePerm) // ignore if dir already exists

	f, err := os.Create(filepath.Join(".", "tmp", "assets.json"))
	check(err)
	defer f.Close()

	n, err := f.Write(marshAssets)
	fmt.Printf("wrote %d bytes\n", n)
	f.Sync()
}
