package horizon

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/stellar/go/protocols/stellarcore"
	"github.com/stellar/go/support/clock"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/log"
)

type stellarCoreClient interface {
	Info(ctx context.Context) (*stellarcore.InfoResponse, error)
}

type healthCache struct {
	response   healthResponse
	lastUpdate time.Time
	ttl        time.Duration
	clock      clock.Clock
	lock       sync.RWMutex
}

func (h *healthCache) get() (healthResponse, bool) {
	h.lock.RLock()
	defer h.lock.RUnlock()

	if h.clock.Now().Sub(h.lastUpdate) > h.ttl {
		return healthResponse{}, false
	}
	return h.response, true
}

func (h *healthCache) set(response healthResponse) {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.lastUpdate = h.clock.Now()
	h.response = response
}

func newHealthCache(ttl time.Duration) *healthCache {
	return &healthCache{ttl: ttl}
}

type healthCheck struct {
	session db.SessionInterface
	ctx     context.Context
	core    stellarCoreClient
	cache   *healthCache
}

type healthResponse struct {
	DatabaseConnected bool `json:"database_connected"`
	CoreUp            bool `json:"core_up"`
	CoreSynced        bool `json:"core_synced"`
}

func (h healthCheck) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	response, ok := h.cache.get()
	if !ok {
		response = healthResponse{
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
		h.cache.set(response)
	}

	if !response.DatabaseConnected || !response.CoreSynced || !response.CoreUp {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}
