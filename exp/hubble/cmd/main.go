// +build go1.13

package main

import (
	"flag"
	"fmt"

	"github.com/stellar/go/exp/hubble"
)

// If no configuration settings are provided, the default is that
// the user is running a standard local ElasticSearch setup.
const elasticSearchDefaultUrl = "http://127.0.0.1:9200"

// Set a default generic index for ElasticSearch.
const elasticSearchDefaultIndex = "testindex"

func main() {
	esUrlPtr := flag.String("esurl", elasticSearchDefaultUrl, "URL of running ElasticSearch server")
	esIndexPtr := flag.String("esindex", elasticSearchDefaultIndex, "index for ElasticSearch")
	flag.Parse()
	fmt.Println("Running the pipeline to serialize XDR entries...")
	session, err := hubble.NewStatePipelineSession(*esUrlPtr, *esIndexPtr)
	if err != nil {
		panic(err)
	}
	err = session.Run()
	if err != nil {
		panic(err)
	}
}
