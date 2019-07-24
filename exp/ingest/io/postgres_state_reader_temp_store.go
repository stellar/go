package io

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
)

// PostgresStateReaderTempStore is a postgres implementation of
// StateReaderTempStore. It's much slower than MemoryStateReaderTempStore
// but has no memory requirements.
type PostgresStateReaderTempStore struct {
	DSN string

	session   *db.Session
	tableName string
}

func (s *PostgresStateReaderTempStore) Open() error {
	var err error
	s.session, err = db.Open("postgres", s.DSN)
	if err != nil {
		return err
	}

	// Begin transaction - without it `ON COMMIT DROP` won't work.
	err = s.session.Begin()
	if err != nil {
		return err
	}

	// Create a temporary table name
	r := make([]byte, 32)
	_, err = rand.Read(r)
	if err != nil {
		return err
	}

	s.tableName = fmt.Sprintf("exp_state_reader_%s", hex.EncodeToString(r))

	// Create table
	_, err = s.session.ExecRaw(fmt.Sprintf(`
	CREATE TEMPORARY TABLE %s (
	    key character varying(255) NOT NULL,
	    value boolean NOT NULL,
	    PRIMARY KEY (key)
	) ON COMMIT DROP;`, s.tableName))
	if err != nil {
		return errors.Wrap(err, "Error creating table")
	}

	return nil
}

func (s *PostgresStateReaderTempStore) Set(key string, value bool) error {
	query := sq.Insert(s.tableName).
		Columns("key", "value").
		Values(key, value).
		Suffix("ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value")

	_, err := s.session.Exec(query)
	return err
}

func (s *PostgresStateReaderTempStore) Get(key string) (bool, error) {
	query := sq.Select("value").
		From(s.tableName).
		Where("key = ?", key)

	var value bool
	if err := s.session.Get(&value, query); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.Wrap(err, "could not get value")
	}

	return value, nil
}

func (s *PostgresStateReaderTempStore) Close() error {
	// This will also drop temp table
	return s.session.Close()
}
