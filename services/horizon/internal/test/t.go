package test

import (
	"io"
	"testing"

	"encoding/json"

	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/services/horizon/internal/db2/schema"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/operationfeestats"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/render/hal"
)

// CoreSession returns a db.Session instance pointing at the stellar core test database
func (t *T) CoreSession() *db.Session {
	return &db.Session{
		DB:  t.CoreDB,
		Ctx: t.Ctx,
	}
}

// Finish finishes the test, logging any accumulated horizon logs to the logs
// output
func (t *T) Finish() {
	RestoreLogger()
	// Reset cached ledger state
	ledger.SetState(ledger.State{})
	operationfeestats.ResetState()

	if t.LogBuffer.Len() > 0 {
		t.T.Log("\n" + t.LogBuffer.String())
	}
}

// HorizonSession returns a db.Session instance pointing at the horizon test
// database
func (t *T) HorizonSession() *db.Session {
	return &db.Session{
		DB:  t.HorizonDB,
		Ctx: t.Ctx,
	}
}

// Scenario loads the named sql scenario into the database
func (t *T) Scenario(name string) *T {
	clearHorizonDB(t.T, t.HorizonDB)
	LoadScenario(name)
	t.UpdateLedgerState()
	return t
}

// ScenarioWithoutHorizon loads the named sql scenario into the database
func (t *T) ScenarioWithoutHorizon(name string) *T {
	LoadScenarioWithoutHorizon(name)
	ResetHorizonDB(t.T, t.HorizonDB)
	t.UpdateLedgerState()
	return t
}

// ResetHorizonDB sets up a new horizon database with empty tables
func ResetHorizonDB(t *testing.T, db *sqlx.DB) {
	clearHorizonDB(t, db)
	_, err := schema.Migrate(db.DB, schema.MigrateUp, 0)
	if err != nil {
		t.Fatalf("could not run migrations up on test db: %v", err)
	}
}

func clearHorizonDB(t *testing.T, db *sqlx.DB) {
	_, err := schema.Migrate(db.DB, schema.MigrateDown, 0)
	if err != nil {
		t.Fatalf("could not run migrations down on test db: %v", err)
	}
}

// UnmarshalPage populates dest with the records contained in the json-encoded page in r
func (t *T) UnmarshalPage(r io.Reader, dest interface{}) hal.Links {
	var env struct {
		Embedded struct {
			Records json.RawMessage `json:"records"`
		} `json:"_embedded"`
		Links struct {
			Self hal.Link `json:"self"`
			Next hal.Link `json:"next"`
			Prev hal.Link `json:"prev"`
		} `json:"_links"`
	}

	err := json.NewDecoder(r).Decode(&env)
	t.Require.NoError(err, "failed to decode page")

	err = json.Unmarshal(env.Embedded.Records, dest)
	t.Require.NoError(err, "failed to decode records")

	return env.Links
}

// UnmarshalNext extracts and returns the next link
func (t *T) UnmarshalNext(r io.Reader) string {
	var env struct {
		Links struct {
			Next struct {
				Href string `json:"href"`
			} `json:"next"`
		} `json:"_links"`
	}

	err := json.NewDecoder(r).Decode(&env)
	t.Require.NoError(err, "failed to decode page")
	return env.Links.Next.Href
}

// UnmarshalExtras extracts and returns extras content
func (t *T) UnmarshalExtras(r io.Reader) map[string]string {
	var resp struct {
		Extras map[string]string `json:"extras"`
	}

	err := json.NewDecoder(r).Decode(&resp)
	t.Require.NoError(err, "failed to decode page")

	return resp.Extras
}

// UpdateLedgerState updates the cached ledger state (or panicing on failure).
func (t *T) UpdateLedgerState() {
	var next ledger.State

	err := t.CoreSession().GetRaw(&next, `
		SELECT
			COALESCE(MAX(ledgerseq), 0) as core_latest
		FROM ledgerheaders
	`)

	if err != nil {
		panic(err)
	}

	err = t.HorizonSession().GetRaw(&next, `
			SELECT
				COALESCE(MIN(sequence), 0) as history_elder,
				COALESCE(MAX(sequence), 0) as history_latest
			FROM history_ledgers
		`)

	if err != nil {
		panic(err)
	}

	ledger.SetState(next)
}
