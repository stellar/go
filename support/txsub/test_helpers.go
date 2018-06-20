package txsub

// This file provides mock implementations for the txsub interfaces
// which are useful in a testing context.
//
// NOTE:  this file is not a test file so that other packages may import
// txsub and use these mocks in their own tests

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/sirupsen/logrus"
	"github.com/stellar/go/support/log"
)

// MockSubmitter is a test helper that simplements the Submitter interface
type MockSubmitter struct {
	R              SubmissionResult
	WasSubmittedTo bool
}

// Submit implements `txsub.Submitter`
func (sub *MockSubmitter) Submit(ctx context.Context, env string) SubmissionResult {
	sub.WasSubmittedTo = true
	return sub.R
}

// MockResultProvider is a test helper that simplements the ResultProvider
// interface
type MockResultProvider struct {
	Results []Result
}

// ResultByHash implements `txsub.ResultProvider`
func (results *MockResultProvider) ResultByHash(ctx context.Context, hash string) (r Result) {
	if len(results.Results) > 0 {
		r = results.Results[0]
		results.Results = results.Results[1:]
	} else {
		r = Result{Err: ErrNoResults}
	}

	return
}

// MockSequenceProvider is a test helper that simplements the SequenceProvider
// interface
type MockSequenceProvider struct {
	Results map[string]uint64
	Err     error
}

// Get implements `txsub.SequenceProvider`
func (results *MockSequenceProvider) Get(addresses []string) (map[string]uint64, error) {
	return results.Results, results.Err
}

// StaticMockServer is a test helper that records it's last request
type StaticMockServer struct {
	*httptest.Server
	LastRequest *http.Request
}

func NewStaticMockServer(response string) *StaticMockServer {
	result := &StaticMockServer{}
	result.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result.LastRequest = r
		fmt.Fprintln(w, response)
	}))

	return result
}

func NewTestContext() context.Context {
	testLogger := log.New()
	testLogger.Entry.Logger.Formatter.(*logrus.TextFormatter).DisableColors = true
	testLogger.Entry.Logger.Level = logrus.DebugLevel
	return log.Set(context.Background(), testLogger)
}
