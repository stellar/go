package db

import (
	"database/sql"

	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
)

const (
	authorizedTransactionTableName = "authorized_transaction"
	allowedFITableName             = "allowed_fi"
	allowedUserTableName           = "allowed_user"
)

func (d *PostgresDatabase) Open(dsn string) error {
	var err error
	d.session, err = db.Open("postgres", dsn)
	if err != nil {
		return err
	}

	return nil
}

func (d *PostgresDatabase) getTable(name string, session *db.Session) *db.Table {
	if session == nil {
		session = d.session
	}

	return &db.Table{
		Name:    name,
		Session: session,
	}
}

// GetAuthorizedTransactionByMemo returns authorized transaction searching by memo
func (r Repository) GetAuthorizedTransactionByMemo(memo string) (*AuthorizedTransaction, error) {
	authorizedTransactionTable := d.getTable(authorizedTransactionTableName, nil)
	var authorizedTransaction AuthorizedTransaction
	err := authorizedTransactionTable.Get(&authorizedTransaction, map[string]interface{}{"memo": memo}).Exec()
	if err != nil {
		switch errors.Cause(err) {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, errors.Wrap(err, "Error getting authorized transaction by memo")
		}
	}

	return &authorizedTransaction, nil
}

// GetAllowedFiByDomain returns allowed FI by a domain
func (r Repository) GetAllowedFiByDomain(domain string) (*AllowedFI, error) {
	allowedFITable := d.getTable(allowedFITableName, nil)
	var allowedFI AllowedFI
	err := allowedFITable.Get(&allowedFI, map[string]interface{}{"domain": domain}).Exec()
	if err != nil {
		switch errors.Cause(err) {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, errors.Wrap(err, "Error getting allowed FI by domain")
		}
	}

	return &allowedFI, nil
}

// GetAllowedUserByDomainAndUserID returns allowed user by domain and userID
func (r Repository) GetAllowedUserByDomainAndUserID(domain, userID string) (*AllowedUser, error) {
	allowedUserTable := d.getTable(allowedUserTableName, nil)
	var allowedUser AllowedUser
	err := allowedUserTable.Get(&allowedUser, map[string]interface{}{"fi_domain": domain, "user_id": userID}).Exec()
	if err != nil {
		switch errors.Cause(err) {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, errors.Wrap(err, "Error getting allowed user by domain and user ID")
		}
	}

	return &allowedUser, nil
}
