package scenarios

import (
	"bytes"
	"io/ioutil"
	"log"
	"os/exec"
)

//go:generate go run assets_generate.go

// Load executes the sql script at `path` on postgres database at `url`
func Load(url string, path string) {
	file, err := assets.Open(path)
	if err != nil {
		log.Panic(err)
	}
	sql, err := ioutil.ReadAll(file)
	if err != nil {
		log.Panic(err)
	}

	reader := bytes.NewReader(sql)
	cmd := exec.Command("psql", url)
	cmd.Stdin = reader

	err = cmd.Run()

	if err != nil {
		log.Panic(err)
	}

}
