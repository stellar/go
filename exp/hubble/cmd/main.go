package main

import (
	"fmt"

	"github.com/stellar/go/exp/hubble"
)

func main() {
	fmt.Println("Running the pipeline to serialize XDR entries...")
	session, err := hubble.NewStatePipelineSession()
	if err != nil {
		panic(err)
	}
	session.Run()
}
