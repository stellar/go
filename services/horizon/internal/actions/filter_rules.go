package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	hProtocol "github.com/stellar/go/protocols/horizon"
	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/render/problem"
)

type QueryPathParams struct {
	Name string `schema:"filter_name" valid:"optional"`
}

type UpdatePathParams struct {
	Name string `schema:"filter_name" valid:"required"`
}

type IngestionFilterHandler struct{}

func (handler IngestionFilterHandler) Get(w http.ResponseWriter, r *http.Request) {
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

	if pp.Name != "" {
		responsePayload, err = handler.findOne(pp.Name, historyQ, r.Context())
		if historyQ.NoRows(err) {
			err = problem.NotFound
		}

	} else {
		responsePayload, err = handler.findAll(historyQ, r.Context())
	}

	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}

	enc := json.NewEncoder(w)
	if err = enc.Encode(responsePayload); err != nil {
		problem.Render(r.Context(), w, err)
	}
}

func (handler IngestionFilterHandler) Update(w http.ResponseWriter, r *http.Request) {
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

	if pp.Name != filterRequest.Name {
		p := problem.BadRequest
		p.Extras = map[string]interface{}{
			"reason": fmt.Sprintf("url path %v, does not match body value %v", pp.Name, filterRequest.Name),
		}
		problem.Render(r.Context(), w, p)
		return
	}

	if err = handler.update(filterRequest, historyQ, r.Context()); err != nil {
		if historyQ.NoRows(err) {
			err = problem.NotFound
		}
		problem.Render(r.Context(), w, err)
	}
}

func (handler IngestionFilterHandler) requestedFilter(r *http.Request) (hProtocol.IngestionFilter, error) {
	var filterRequest hProtocol.IngestionFilter
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&filterRequest); err != nil {
		p := problem.BadRequest
		p.Extras = map[string]interface{}{
			"reason": fmt.Sprintf("invalid json for filter config %v", err.Error()),
		}
		return hProtocol.IngestionFilter{}, p
	}
	return filterRequest, nil
}

func (handler IngestionFilterHandler) update(filterRequest hProtocol.IngestionFilter, historyQ *history.Q, ctx context.Context) error {
	//TODO, consider type specific schema validation of the json in filterRequest.Rules based on filterRequest.Name
	// if name='asset', verify against an Asset Config Struct
	// if name='account', verify against an Account Config Struct
	filterConfig := history.FilterConfig{}
	filterConfig.Enabled = filterRequest.Enabled
	filterConfig.Name = filterRequest.Name

	filterRules, err := json.Marshal(filterRequest.Rules)
	if err != nil {
		p := problem.ServerError
		p.Extras = map[string]interface{}{
			"reason": fmt.Sprintf("unable to serialize filter rules resource from json %v", err.Error()),
		}
		return p
	}
	filterConfig.Rules = string(filterRules)
	return historyQ.UpdateFilterConfig(ctx, filterConfig)
}

func (handler IngestionFilterHandler) findOne(name string, historyQ *history.Q, ctx context.Context) (hProtocol.IngestionFilter, error) {
	filter, err := historyQ.GetFilterByName(ctx, name)
	if err != nil {
		return hProtocol.IngestionFilter{}, err
	}

	rules, err := handler.rules(filter.Rules)
	if err != nil {
		return hProtocol.IngestionFilter{}, err
	}
	return handler.resource(filter, rules), nil
}

func (handler IngestionFilterHandler) findAll(historyQ *history.Q, ctx context.Context) ([]hProtocol.IngestionFilter, error) {
	configs, err := historyQ.GetAllFilters(ctx)
	if err != nil {
		return nil, err
	}
	resources := []hProtocol.IngestionFilter{}
	for _, config := range configs {
		rules, err := handler.rules(config.Rules)
		if err != nil {
			return nil, err
		}
		resources = append(resources, handler.resource(config, rules))
	}
	return resources, nil
}

func (handler IngestionFilterHandler) rules(input string) (map[string]interface{}, error) {
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

func (handler IngestionFilterHandler) resource(config history.FilterConfig, rules map[string]interface{}) hProtocol.IngestionFilter {
	return hProtocol.IngestionFilter{
		Rules:        rules,
		Enabled:      config.Enabled,
		Name:         config.Name,
		LastModified: config.LastModified,
	}
}
