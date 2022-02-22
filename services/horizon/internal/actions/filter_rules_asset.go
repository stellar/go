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

type assetFilterResource struct {
	Rules   filters.AssetFilterRules `json:"rules"`
	Enabled bool                     `json:"enabled"`
	Name    string                   `json:"name"`
}

func (afr assetFilterResource) Validate() error {
	for _, asset := range afr.Rules.CanonicalWhitelist {
		if !isAsset(asset) {
			return fmt.Errorf("%q is not a valid asset issuer:code", asset)
		}
	}
	return nil
}

func (afr *assetFilterResource) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, afr); err != nil {
		return err
	}
	return afr.Validate()
}

type AssetFilterRuleHandler struct{}

func (handler AssetFilterRuleHandler) Get(w http.ResponseWriter, r *http.Request) {
	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}
	filter, err := historyQ.GetFilterByName(r.Context(), history.FilterAssetFilterName)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}

	var assetFilterRules = filters.AssetFilterRules{}
	if err = json.Unmarshal([]byte(filter.Rules), &assetFilterRules); err != nil {
		p := problem.ServerError
		p.Extras = map[string]interface{}{
			"reason": "invalid asset filter rule json in db",
		}
		problem.Render(r.Context(), w, err)
		return
	}

	assetFilterResource := &assetFilterResource{
		Rules:   assetFilterRules,
		Enabled: filter.Enabled,
		Name:    filter.Name,
	}

	enc := json.NewEncoder(w)
	if err = enc.Encode(assetFilterResource); err != nil {
		problem.Render(r.Context(), w, err)
	}
}

func (handler AssetFilterRuleHandler) Set(w http.ResponseWriter, r *http.Request) {
	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}
	var assetFilterRequest assetFilterResource
	dec := json.NewDecoder(r.Body)
	if err = dec.Decode(&assetFilterRequest); err != nil {
		p := problem.BadRequest
		p.Extras = map[string]interface{}{
			"reason": err.Error(),
		}
		problem.Render(r.Context(), w, err)
		return
	}

	var filterConfig history.FilterConfig
	var assetFilterRules []byte
	filterConfig.Enabled = assetFilterRequest.Enabled
	filterConfig.Name = history.FilterAssetFilterName

	if assetFilterRules, err = json.Marshal(assetFilterRequest.Rules); err != nil {
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
