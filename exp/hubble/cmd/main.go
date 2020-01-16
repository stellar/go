// +build go1.13

package main

import (
	"flag"
	"fmt"

	"github.com/stellar/go/exp/hubble"
	"github.com/stellar/go/support/errors"
)

// If no configuration settings are provided, the default is that
// the user is running a standard local ElasticSearch setup.
const elasticSearchDefaultURL = "http://127.0.0.1:9200"

// Set a default generic index for ElasticSearch.
const elasticSearchDefaultIndex = "testindex"

func main() {
	// TODO: Remove pipelineTypePtr flag once current state pipeline and elastic search are merged.
	typeFlagStr := fmt.Sprintf("type of state pipeline, choices are %s and %s", hubble.PipelineDefaultType, hubble.PipelineSearchType)
	pipelineTypePtr := flag.String("type", hubble.PipelineDefaultType, typeFlagStr)
	esURLPtr := flag.String("esurl", elasticSearchDefaultURL, "URL of running ElasticSearch server")
	esIndexPtr := flag.String("esindex", elasticSearchDefaultIndex, "index for ElasticSearch")
	flag.Parse()

	pipelineType := *pipelineTypePtr
	// Validate that pipeline type is either "current" or "search".
	if (pipelineType != hubble.PipelineDefaultType) && (pipelineType != hubble.PipelineSearchType) {
		panic(errors.Errorf("invalid pipeline type %s, must be '%s' or '%s'", pipelineType, hubble.PipelineDefaultType, hubble.PipelineSearchType))
	}

	session, err := hubble.NewStatePipelineSession(pipelineType, *esURLPtr, *esIndexPtr)
	if err != nil {
		panic(errors.Wrap(err, "could not make new state pipeline session"))
	}
	fmt.Printf("Running state pipeline session of type %s\n", pipelineType)
	err = session.Run()
	if err != nil {
		panic(errors.Wrap(err, "could not run session"))
	}
}
