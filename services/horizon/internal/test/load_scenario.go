package test

import (
	"github.com/stellar/go/services/horizon/internal/test/scenarios"
)

func (t *T) loadScenario(scenarioName string, includeHorizon bool) {
	stellarCorePath := scenarioName + "-core.sql"

	scenarios.Load(t.CoreDB, StellarCoreDatabaseURL(), stellarCorePath)

	if includeHorizon {
		horizonPath := scenarioName + "-horizon.sql"
		scenarios.Load(t.HorizonDB, DatabaseURL(), horizonPath)
	}
}
