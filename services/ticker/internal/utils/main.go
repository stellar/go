package utils

import (
	"fmt"
	"os"
	"time"
)

// PanicIfError is an utility function that panics if err != nil
func PanicIfError(e error) {
	if e != nil {
		panic(e)
	}
}

// WriteJSONToFile wrtites a json []byte dump to <filename>
func WriteJSONToFile(jsonBytes []byte, filename string) (numBytes int, err error) {
	f, err := os.Create(filename)
	PanicIfError(err)
	defer f.Close()

	numBytes, err = f.Write(jsonBytes)
	if err != nil {
		return
	}

	err = f.Sync()
	if err != nil {
		return
	}

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

// TimeToTimestamp converts a time.Time into a Unix epoch
func TimeToUnixEpoch(t time.Time) int64 {
	return t.UnixNano() / 1000000
}

// TimeToRFC3339 converts a time.Time to a string in RFC3339 format
func TimeToRFC3339(t time.Time) string {
	return t.Format(time.RFC3339)
}

// CalcSpread calculates the spread stats for the given bidMax and askMin orderbook values
func CalcSpread(bidMax float64, askMin float64) (spread float64, midPoint float64) {
	if askMin == 0 || bidMax == 0 {
		return 0, 0
	}
	spread = (askMin - bidMax) / askMin
	midPoint = bidMax + spread/2.0
	return
}
