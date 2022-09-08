package actions

import (
	"encoding/json"
	"net/http"

	"github.com/stellar/go/support/log"
	supportProblem "github.com/stellar/go/support/render/problem"
)

type RootResponse struct {
	Version      string `json:"version"`
	LedgerSource string `json:"ledger_source"`
	IndexSource  string `json:"index_source"`
	LatestLedger uint32 `json:"latest_indexed_ledger"`
}

func Root(config RootResponse) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/hal+json; charset=utf-8")
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		err := encoder.Encode(config)
		if err != nil {
			log.Error(err)
			sendErrorResponse(r.Context(), w, supportProblem.ServerError)
		}
	}
}
