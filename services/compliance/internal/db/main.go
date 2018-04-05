package db

import (
	"time"

	"github.com/stellar/go/support/db"
)

type Database interface {
	GetAuthorizedTransactionByMemo(memo string) (*entities.AuthorizedTransaction, error)
	GetAllowedFiByDomain(domain string) (*entities.AllowedFi, error)
	GetAllowedUserByDomainAndUserID(domain, userID string) (*entities.AllowedUser, error)
}

type PostgresDatabase struct {
	session *db.Session
}

// AllowedFI represents allowed FI
type AllowedFI struct {
	exists    bool
	ID        *int64    `db:"id"`
	Name      string    `db:"name"`
	Domain    string    `db:"domain"`
	PublicKey string    `db:"public_key"`
	AllowedAt time.Time `db:"allowed_at"`
}

// AllowedUser represents allowed user
type AllowedUser struct {
	exists      bool
	ID          *int64    `db:"id"`
	FiName      string    `db:"fi_name"`
	FiDomain    string    `db:"fi_domain"`
	FiPublicKey string    `db:"fi_public_key"`
	UserID      string    `db:"user_id"`
	AllowedAt   time.Time `db:"allowed_at"`
}

// AuthorizedTransaction represents authorized transaction
type AuthorizedTransaction struct {
	exists         bool
	ID             *int64    `db:"id"`
	TransactionID  string    `db:"transaction_id"`
	Memo           string    `db:"memo"`
	TransactionXdr string    `db:"transaction_xdr"`
	AuthorizedAt   time.Time `db:"authorized_at"`
	Data           string    `db:"data"`
}
