package txsub

import (
	"mime"
	"net/http"
	"sync"

	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/hal"
	"github.com/stellar/go/support/render/problem"
)

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hr := HandlerResponse{}

	// Validate body type
	c := r.Header.Get("Content-Type")
	if c != "" {
		mt, _, err := mime.ParseMediaType(c)

		if err != nil {
			hr.Err = err
		}

		if mt != "application/x-www-form-urlencoded" && mt != "multipart/form-data" {
			hr.Err = problem.P{
				Type:   "unsupported_media_type",
				Title:  "Unsupported Media Type",
				Status: http.StatusUnsupportedMediaType,
				Detail: "The request has an unsupported content type. Presently, the " +
					"only supported content type is application/x-www-form-urlencoded.",
			}
		}
	}

	tx := r.FormValue("tx")

	submission := h.Driver.SubmitTransaction(r.Context(), tx)
	select {
	case result := <-submission:
		hr.Result = result

	case <-r.Context().Done():
		hr.Err = &problem.P{
			Type:   "timeout",
			Title:  "Timeout",
			Status: http.StatusGatewayTimeout,
			Detail: "Your request timed out before completing.  Please try your " +
				"request again.",
		}
	}

	hr.LoadResource()

	if hr.Err != nil {
		hal.Render(w, hr.Err)
	} else {
		hal.Render(w, hr.Resource)
	}
}

// Run is the function that runs in the background that triggers Tick each
// second.
func (h *Handler) Run() {
	for {
		select {
		case <-h.Ticks.C:
			h.tick()
		case <-h.Context.Done():
			log.Info("finished background ticker")
			return
		}
	}
}

// Tick triggers txsub to update all of it's background processes.
func (h *Handler) tick() {
	var wg sync.WaitGroup
	log.Debug("ticking app")

	wg.Add(1)
	go func() { h.Driver.Tick(h.Context); wg.Done() }()
	wg.Wait()

	// finally, update metrics
	log.Debug("finished ticking app")
}
