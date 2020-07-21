package horizon

import (
	"net/http"
)

func (action RootAction) Handle(w http.ResponseWriter, r *http.Request) {
	ap := &action.Action
	ap.Prepare(w, r)
	ap.Execute(&action)
}

func (action TradeAggregateIndexAction) Handle(w http.ResponseWriter, r *http.Request) {
	ap := &action.Action
	ap.Prepare(w, r)
	ap.Execute(&action)
}

func (action TradeIndexAction) Handle(w http.ResponseWriter, r *http.Request) {
	ap := &action.Action
	ap.Prepare(w, r)
	ap.Execute(&action)
}

func (action TransactionCreateAction) Handle(w http.ResponseWriter, r *http.Request) {
	ap := &action.Action
	ap.Prepare(w, r)
	ap.Execute(&action)
}
