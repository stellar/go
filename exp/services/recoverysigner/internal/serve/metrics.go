package serve

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stellar/go/exp/services/recoverysigner/internal/account"
	supportlog "github.com/stellar/go/support/log"
)

type metricsHandler struct {
	Logger       *supportlog.Entry
	AccountStore account.Store
	Namespace    string
}

func (m metricsHandler) Registry() *prometheus.Registry {
	reg := prometheus.NewRegistry()

	err := reg.Register(metricAccountsCount{
		Logger:       m.Logger,
		Namespace:    m.Namespace,
		AccountStore: m.AccountStore,
	}.NewCollector())
	if err != nil {
		m.Logger.Warn("Error registering metric for accounts count: ", err)
	}

	return reg
}
