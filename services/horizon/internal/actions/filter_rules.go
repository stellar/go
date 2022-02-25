package actions

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/filters"
	"github.com/stellar/go/support/render/problem"
)

// standard resource interface for a filter config
type filterResource struct {
	Rules        map[string]interface{} `json:"rules"`
	Enabled      bool                   `json:"enabled"`
	Name         string                 `json:"name,omitempty"`
	LastModified int64                  `json:"last_modified,omitempty"`
}

type QueryPathParams struct {
	NAME string `schema:"name" valid:"optional"`
}

type UpdatePathParams struct {
	NAME string `schema:"name" valid:"required"`
}

type FilterRuleHandler struct{}

func (handler FilterRuleHandler) Get(w http.ResponseWriter, r *http.Request) {
	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}

	pp := QueryPathParams{}
	err = getParams(&pp, r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}

	var responsePayload interface{}

	if pp.NAME != "" {
		responsePayload, err = handler.findOne(pp.NAME, historyQ, r.Context())
	} else {
		responsePayload, err = handler.findAll(historyQ, r.Context())
	}

	if err != nil {
		problem.Render(r.Context(), w, err)
	}

	enc := json.NewEncoder(w)
	if err = enc.Encode(responsePayload); err != nil {
		problem.Render(r.Context(), w, err)
	}
}

func (handler FilterRuleHandler) Create(w http.ResponseWriter, r *http.Request) {
	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}

	filterRequest, err := handler.requestedFilter(r)
	if err != nil {
		problem.Render(r.Context(), w, err)
	}

	existing, err := handler.findOne(filterRequest.Name, historyQ, r.Context())
	if err != sql.ErrNoRows {
		if existing != nil {
			err := problem.BadRequest
			err.Extras = map[string]interface{}{
				"filter already exists": filterRequest.Name,
			}
		}
		problem.Render(r.Context(), w, err)
		return
	}

	if err = handler.upsert(filterRequest, historyQ, r.Context()); err != nil {
		problem.Render(r.Context(), w, err)
	}
	w.WriteHeader(201)
}

func (handler FilterRuleHandler) Update(w http.ResponseWriter, r *http.Request) {
	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}

	pp := &UpdatePathParams{}
	err = getParams(pp, r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}

	filterRequest, err := handler.requestedFilter(r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}

	if pp.NAME != filterRequest.Name {
		p := problem.BadRequest
		p.Extras = map[string]interface{}{
			"reason": fmt.Sprintf("url path %v, does not match body value %v", pp.NAME, filterRequest.Name),
		}
		problem.Render(r.Context(), w, p)
		return
	}

	if _, err = handler.findOne(filterRequest.Name, historyQ, r.Context()); err != nil {
		// not found or other error
		problem.Render(r.Context(), w, err)
	}

	if err = handler.upsert(filterRequest, historyQ, r.Context()); err != nil {
		problem.Render(r.Context(), w, err)
	}
}

func (handler FilterRuleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}

	pp := &UpdatePathParams{}
	err = getParams(pp, r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}

	if _, err = handler.findOne(pp.NAME, historyQ, r.Context()); err != nil {
		// not found or other error
		problem.Render(r.Context(), w, err)
	}

	if err = historyQ.DeleteFilterByName(r.Context(), pp.NAME); err != nil {
		problem.Render(r.Context(), w, err)
	}
	w.WriteHeader(204)
}

func (handler FilterRuleHandler) requestedFilter(r *http.Request) (*filterResource, error) {
	filterRequest := &filterResource{}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(filterRequest); err != nil {
		p := problem.BadRequest
		p.Extras = map[string]interface{}{
			"reason": fmt.Sprintf("invalid json for filter config %v", err.Error()),
		}
		return nil, p
	}
	return filterRequest, nil
}

func (handler FilterRuleHandler) upsert(filterRequest *filterResource, historyQ *history.Q, ctx context.Context) error {
	//TODO, consider type specific schema validation of the json in filterRequest.Rules based on filterRequest.Name
	// if name='asset', verify against an Asset Config Struct
	// if name='account', verify against an Account Config Struct
	filterConfig := history.FilterConfig{}
	filterConfig.Enabled = filterRequest.Enabled
	filterConfig.Name = filterRequest.Name

	if !filters.SupportedFilterNames(filterRequest.Name) {
		p := problem.ServerError
		p.Extras = map[string]interface{}{
			"reason": fmt.Sprintf("invalid filter name, %v, no implementation for this exists", filterRequest.Name),
		}
		return p
	}

	filterRules, err := json.Marshal(filterRequest.Rules)
	if err != nil {
		p := problem.ServerError
		p.Extras = map[string]interface{}{
			"reason": fmt.Sprintf("unable to serialize filter rules resource from json %v", err.Error()),
		}
		return p
	}
	filterConfig.Rules = string(filterRules)
	return historyQ.UpsertFilterConfig(ctx, filterConfig)
}

func (handler FilterRuleHandler) findOne(name string, historyQ *history.Q, ctx context.Context) (*filterResource, error) {
	filter, err := historyQ.GetFilterByName(ctx, name)
	if err != nil {
		return nil, err
	}
	rules, err := handler.rules(filter.Rules)
	if err != nil {
		return nil, err
	}
	return handler.resource(filter, rules), nil
}

func (handler FilterRuleHandler) findAll(historyQ *history.Q, ctx context.Context) ([]*filterResource, error) {
	configs, err := historyQ.GetAllFilters(ctx)
	if err != nil {
		return nil, err
	}
	resources := []*filterResource{}
	for _, config := range configs {
		rules, err := handler.rules(config.Rules)
		if err != nil {
			return nil, err
		}
		resources = append(resources, handler.resource(config, rules))
	}
	return resources, nil
}

func (handler FilterRuleHandler) rules(input string) (map[string]interface{}, error) {
	rules := make(map[string]interface{})
	if err := json.Unmarshal([]byte(input), &rules); err != nil {
		p := problem.ServerError
		p.Extras = map[string]interface{}{
			"reason": "invalid filter rule json in db",
		}
		return nil, p
	}
	return rules, nil
}

func (handler FilterRuleHandler) resource(config history.FilterConfig, rules map[string]interface{}) *filterResource {
	return &filterResource{
		Rules:        rules,
		Enabled:      config.Enabled,
		Name:         config.Name,
		LastModified: config.LastModified,
	}
}
