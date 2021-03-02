package horizon

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/stellar/go/protocols/stellarcore"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/log"
)

type stellarCoreClient interface {
	Info(ctx context.Context) (*stellarcore.InfoResponse, error)
}

type healthCheck struct {
	session db.SessionInterface
	ctx     context.Context
	core    stellarCoreClient
}

type healthResponse struct {
	DatabaseConnected bool `json:"database_connected"`
	CoreUp            bool `json:"core_up"`
	CoreSynced        bool `json:"core_synced"`
}

func (h healthCheck) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	response := healthResponse{
		DatabaseConnected: true,
		CoreUp:            true,
		CoreSynced:        true,
	}
	if err := h.session.Ping(); err != nil {
		log.Warnf("could ping db: %s", err)
		response.DatabaseConnected = false
	}
	if resp, err := h.core.Info(h.ctx); err != nil {
		log.Warnf("request to stellar core failed: %s", err)
		response.CoreUp = false
		response.CoreSynced = false
	} else {
		response.CoreSynced = resp.IsSynced()
	}

	if !response.DatabaseConnected || !response.CoreSynced || !response.CoreUp {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}
