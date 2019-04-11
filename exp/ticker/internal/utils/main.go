package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// PanicIfError is an utility function that panics if err != nil
func PanicIfError(e error) {
	if e != nil {
		panic(e)
	}
}

// WriteJSONToFile wrtites a json []byte dump to .tmp/<filename>
func WriteJSONToFile(jsonBytes []byte, filename string) (numBytes int, err error) {
	path := filepath.Join(".", "tmp")
	_ = os.Mkdir(path, os.ModePerm) // ignore if dir already exists

	f, err := os.Create(filepath.Join(".", "tmp", filename))
	PanicIfError(err)
	defer f.Close()

	numBytes, err = f.Write(jsonBytes)
	f.Sync()

	return
}

// SliceDiff returns the elements in `a` that aren't in `b`.
func SliceDiff(a, b []string) (diff []string) {
	bmap := map[string]bool{}
	for _, x := range b {
		bmap[x] = true
	}
	for _, x := range a {
		if _, ok := bmap[x]; !ok {
			diff = append(diff, x)
		}
	}
	return
}

// GetAssetString returns a string representation of an asset
func GetAssetString(assetType string, code string, issuer string) string {
	if assetType == "native" {
		return "native"
	}
	return fmt.Sprintf("%s:%s", code, issuer)
}
