package actions

import (
	"context"
	"encoding/json"
	"net/http"

	horizonContext "github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/render/problem"
)

// standard resource interface for a filter config
type filterResource struct {
	Rules   map[string]interface{}   `json:"rules"`
	Enabled bool                     `json:"enabled"`
	Name    string                   `json:"name"`
	LastModified int64               `json:"last_modified,omitempty"`
}

type FilterQuery struct {
	NAME string `schema:"name" valid:"optional"`
}

type FilterRuleHandler struct{}

func (handler FilterRuleHandler) Get(w http.ResponseWriter, r *http.Request) {
	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}

    qp := FilterQuery{}
	err = getParams(&qp, r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}

	var responsePayload interface{}

	if (qp.NAME != "") {
		responsePayload, err = handler.findOne(qp.NAME, historyQ, r.Context())
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

func (handler FilterRuleHandler) Set(w http.ResponseWriter, r *http.Request) {
	historyQ, err := horizonContext.HistoryQFromRequest(r)
	if err != nil {
		problem.Render(r.Context(), w, err)
		return
	}
	var filterRequest filterResource
	dec := json.NewDecoder(r.Body)
	if err = dec.Decode(&filterRequest); err != nil {
		p := problem.BadRequest
		p.Extras = map[string]interface{}{
			"invalid json for filter config": err.Error(),
		}
		problem.Render(r.Context(), w, err)
		return
	}

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
			"reason": "unable to serialize asset filter rules resource from json",
		}
		problem.Render(r.Context(), w, err)
		return
	}
	filterConfig.Rules = string(filterRules)

	if err = historyQ.SetFilterConfig(r.Context(), filterConfig); err != nil {
		problem.Render(r.Context(), w, err)
		return
	}
}

func (handler FilterRuleHandler) findOne(name string, historyQ *history.Q, ctx context.Context) (*filterResource, error) {
	filter, err := historyQ.GetFilterByName(ctx,name)
	if err != nil {
		return nil, err
	}
	rules, err := handler.rules(filter.Rules)
	if err != nil {
		return nil, err
	}
	return handler.resource(filter, rules), nil
}

func (handler FilterRuleHandler) findAll(historyQ *history.Q, ctx context.Context) ([]*filterResource, error){
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

func (handler FilterRuleHandler) resource(config history.FilterConfig, rules map[string]interface{}) *filterResource{
	return &filterResource{
		Rules:   rules,
		Enabled: config.Enabled,
		Name:    config.Name,
		LastModified: config.LastModified,
	}
}
