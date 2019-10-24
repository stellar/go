package main

import (
	"fmt"

	"github.com/stellar/go/exp/hubble"
)

func main() {
	fmt.Println("Running the pipeline to serialize XDR entries...")
	hubble.RunStatePipelineSession()
}
