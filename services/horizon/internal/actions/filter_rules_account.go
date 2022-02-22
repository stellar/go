package actions

import (
	"encoding/json"
	"fmt"
	"net/http"

	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/filters"
	"github.com/stellar/go/support/render/problem"
)

// TODO(fons): this code is identical to that in `filter_rules_asset.go` we should refactor it
type accountFilterResource struct {
	Rules   filters.AccountFilterRules `json:"rules"`
	Enabled bool                       `json:"enabled"`
	Name    string                     `json:"name"`
}

func (afr accountFilterResource) Validate() error {
	for _, account := range afr.Rules.CanonicalWhitelist {
		if !isAccountID(account) {
			return fmt.Errorf("%q is not a valid account issuer:code", account)
		}
	}
	return nil
}

func (afr *accountFilterResource) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, afr); err != nil {
		return err
	}
	return afr.Validate()
}

type AccountFilterRuleHandler struct{}

func (handler AccountFilterRuleHandler) Get(w http.ResponseWriter, r *http.Request) {
	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}
	filter, err := historyQ.GetFilterByName(r.Context(), history.FilterAccountFilterName)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}

	var accountFilterRules = filters.AccountFilterRules{}
	if err = json.Unmarshal([]byte(filter.Rules), &accountFilterRules); err != nil {
		p := problem.ServerError
		p.Extras = map[string]interface{}{
			"reason": "invalid asset filter rule json in db",
		}
		problem.Render(r.Context(), w, err)
		return
	}

	accountFilterResource := &accountFilterResource{
		Rules:   accountFilterRules,
		Enabled: filter.Enabled,
		Name:    filter.Name,
	}

	enc := json.NewEncoder(w)
	if err = enc.Encode(accountFilterResource); err != nil {
		problem.Render(r.Context(), w, err)
	}
}

func (handler AccountFilterRuleHandler) Set(w http.ResponseWriter, r *http.Request) {
	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}
	var accountFilterRequest accountFilterResource
	dec := json.NewDecoder(r.Body)
	if err = dec.Decode(&accountFilterRequest); err != nil {
		p := problem.BadRequest
		p.Extras = map[string]interface{}{
			"reason": err.Error(),
		}
		problem.Render(r.Context(), w, err)
		return
	}

	var filterConfig history.FilterConfig
	var assetFilterRules []byte
	filterConfig.Enabled = accountFilterRequest.Enabled
	filterConfig.Name = history.FilterAssetFilterName

	if assetFilterRules, err = json.Marshal(accountFilterRequest.Rules); err != nil {
		p := problem.ServerError
		p.Extras = map[string]interface{}{
			"reason": "unable to serialize asset filter rules resource from json",
		}
		problem.Render(r.Context(), w, err)
		return
	}
	filterConfig.Rules = string(assetFilterRules)

	if err = historyQ.SetFilterConfig(r.Context(), filterConfig); err != nil {
		problem.Render(r.Context(), w, err)
		return
	}
}
