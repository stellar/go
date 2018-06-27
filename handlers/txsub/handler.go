package txsub

import (
	"mime"
	"net/http"
	"sync"

	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
)

// Run is the function that runs in the background that triggers Tick each
// second.
func (h *Handler) Run() {
	for {
		select {
		case <-h.Ticks.C:
			h.Tick()
		case <-h.Context.Done():
			log.Info("finished background ticker")
			return
		}
	}
}

// Tick triggers txsub to update all of it's background processes.
func (h *Handler) Tick() {
	var wg sync.WaitGroup
	log.Debug("ticking app")

	wg.Add(1)
	go func() { h.Driver.Tick(h.Context); wg.Done() }()
	wg.Wait()

	// finally, update metrics
	log.Debug("finished ticking app")
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Validate body type is `application/x-www-form-urlencoded`
	c := r.Header.Get("Content-Type")
	mt, _, err := mime.ParseMediaType(c)
	if err != nil {
		h.Err = err
	}
	switch {
	case mt == "application/x-www-form-urlencoded":
		break
	case mt == "multipart/form-data":
		break
	default:
		h.Err = problem.P{
			Type:   "unsupported_media_type",
			Title:  "Unsupported Media Type",
			Status: http.StatusUnsupportedMediaType,
			Detail: "The request has an unsupported content type. Presently, the " +
				"only supported content type is application/x-www-form-urlencoded.",
		}
	}
	// 2. Get tx envelope string from form
	tx := r.FormValue("tx")
	// 3. submit tx via submission system
	submission := h.Driver.SubmitTransaction(r.Context(), tx)
	// 4. deal with submission result
	select {
	case result := <-submission:
		h.Result = result
	case <-h.Context.Done():
		h.Err = &problem.P{
			Type:   "timeout",
			Title:  "Timeout",
			Status: http.StatusGatewayTimeout,
			Detail: "Your request timed out before completing.  Please try your " +
				"request again.",
		}
	}
	// 5. load resource
	// 6. write hal
	hal.Render(w, h.Result)
}
