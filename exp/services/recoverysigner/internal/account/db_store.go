package account

import (
	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/support/clock"
)

type DBStore struct {
	DB    *sqlx.DB
	Clock *clock.Clock
}
