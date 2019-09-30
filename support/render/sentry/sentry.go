package sentry

import (
	"context"

	"github.com/getsentry/sentry-go"
)

// AsyncCapture reports err to Sentry asynchronously.
// Note that the Sentry Go SDK itself, provides two built-in transports. HTTPTransport,
// which is non-blocking and is used by default. And HTTPSyncTransport which is
// blocking.
// To use HTTPSyncTransport, you would need to configure Sentry client with ClientOptions
// during initialization.
func AsyncCapture(ctx context.Context, err error) {
	// Use Sentry Scope to capture other information in the context
	// https://docs.sentry.io/enriching-error-data/context/?platform=go#tagging-events
	sentry.CaptureException(err)
}
