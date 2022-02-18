package actions

import (
	"encoding/json"
	"fmt"
	"net/http"

	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/support/render/problem"
)

type accountWhitelist []string

func (wl accountWhitelist) Validate() error {
	for _, account := range wl {
		if !isAccountID(account) {
			return fmt.Errorf("%q is not a valid account strkey", account)
		}
	}
	return nil
}

func (wl *accountWhitelist) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, wl); err != nil {
		return err
	}
	return wl.Validate()
}

type AccountFilterWhitelistHandler struct{}

func (handler AccountFilterWhitelistHandler) Get(w http.ResponseWriter, r *http.Request) {
	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}
	l, err := historyQ.GetAccountFilterWhitelist(r.Context())
	if err != nil {
		problem.Render(r.Context(), w, err)
	}
	enc := json.NewEncoder(w)
	if err = enc.Encode(l); err != nil {
		problem.Render(r.Context(), w, err)
		return
	}
}

func (handler AccountFilterWhitelistHandler) Set(w http.ResponseWriter, r *http.Request) {
	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}
	var wl accountWhitelist
	dec := json.NewDecoder(r.Body)
	if err = dec.Decode(&wl); err != nil {
		p := problem.BadRequest
		p.Extras = map[string]interface{}{
			"reason": err.Error(),
		}
		problem.Render(r.Context(), w, err)
		return
	}
	if err = historyQ.SetAccountFilterWhitelist(r.Context(), wl); err != nil {
		problem.Render(r.Context(), w, err)
		return
	}
}
