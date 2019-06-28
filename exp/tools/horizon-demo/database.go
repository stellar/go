package main

import (
	"database/sql"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/xdr"
)

/* Schema:

CREATE TABLE accounts_signers (
    account character varying(64),
    signer character varying(64)
);

ALTER TABLE ONLY accounts_signers
    ADD CONSTRAINT account_signer_unique UNIQUE (account, signer);

CREATE INDEX signers_index ON accounts_signers USING btree (signer);

*/

type Database struct {
	*db.Session
}

type transactions struct {
	Hash       string `db:"hash"`
	Success    bool   `db:"success"`
	Ledger     int32  `db:"ledger"`
	Source     string `db:"source"`
	FeePaid    uint32 `db:"fee_paid"`
	Operations int    `db:"operations"`
}

type accountSigner struct {
	Account string `db:"account"`
	Signer  string `db:"signer"`
}

func NewDatabase(uri string) (*Database, error) {
	session, err := db.Open("postgres", uri)
	if err != nil {
		return nil, err
	}

	db := &Database{
		Session: session,
	}

	return db, nil
}

// InsertTransaction adds a new transaction to the database. It probably
// shouldn't use `io.LedgerTransaction` but it's just a demo code.
func (d *Database) InsertTransaction(transaction io.LedgerTransaction) (sql.Result, error) {
	row := &transactions{
		Hash:       "",
		Success:    transaction.Result.Result.Result.Code == xdr.TransactionResultCodeTxSuccess,
		Ledger:     1,
		Source:     transaction.Envelope.Tx.SourceAccount.Address(),
		FeePaid:    uint32(transaction.Envelope.Tx.Fee),
		Operations: len(transaction.Envelope.Tx.Operations),
	}
	return d.Session.GetTable("transactions").Insert(row).Exec()
}

// InsertAccountSigner adds a account signer pair to the database.
func (d *Database) InsertAccountSigner(account, signer string) (sql.Result, error) {
	row := &accountSigner{
		Account: account,
		Signer:  signer,
	}
	return d.Session.GetTable("accounts_signers").Insert(row).Exec()
}

// RemoveAccountSigner removes account signer pair.
func (d *Database) RemoveAccountSigner(account, signer string) (sql.Result, error) {
	return d.Session.GetTable("accounts_signers").
		Delete("account = ? && signer = ?", account, signer).Exec()
}
