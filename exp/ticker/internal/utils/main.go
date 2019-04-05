package utils

import (
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
