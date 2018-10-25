package httpx

import (
	"context"
	"net/http"

	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/support/log"
)

func RequestFromContext(ctx context.Context) *http.Request {
	found := ctx.Value(&horizonContext.RequestContextKey)

	if found == nil {
		return nil
	}

	return found.(*http.Request)
}

// RequestContext returns a context representing the provided http action.
// It also integrates `http.CloseNotifier` with `context.Context`, returning a context
// that will be canceled when the http connection underlying `w` is closed.
func RequestContext(parent context.Context, w http.ResponseWriter, r *http.Request) (context.Context, func()) {
	if r == nil {
		panic("Cannot bind nil *http.Request to context tree")
	}

	ctx, cancel := context.WithCancel(parent)
	notifier, ok := w.(http.CloseNotifier)

	var closedByClient <-chan bool

	if ok {
		closedByClient = notifier.CloseNotify()
	} else {
		closedByClient = make(chan bool)
	}

	// listen for the connection to close, trigger cancellation
	go func() {
		select {
		case <-closedByClient:
			log.Ctx(parent).Info("Request closed by client")
			cancel()
		case <-ctx.Done():
			return
		}
	}()

	return context.WithValue(ctx, &horizonContext.RequestContextKey, r), cancel
}
