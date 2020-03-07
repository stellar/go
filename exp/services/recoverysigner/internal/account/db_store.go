package account

import (
	"github.com/jmoiron/sqlx"
)

func NewDBStore(db *sqlx.DB) Store {
	return &dbStore{
		db: db,
	}
}

type dbStore struct {
	db *sqlx.DB
}

func (ms *dbStore) Add(a Account) error {
	return nil
}

func (ms *dbStore) Delete(address string) error {
	return nil
}

func (ms *dbStore) Get(address string) (Account, error) {
	return Account{}, nil
}

func (ms *dbStore) FindWithIdentityAddress(address string) ([]Account, error) {
	return []Account{}, nil
}

func (ms *dbStore) FindWithIdentityPhoneNumber(phoneNumber string) ([]Account, error) {
	return []Account{}, nil
}

func (ms *dbStore) FindWithIdentityEmail(email string) ([]Account, error) {
	return []Account{}, nil
}
