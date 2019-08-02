package io

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
)

// postgresStateReaderTempStoreCacheSize defines the maximum number of
// entries in the cache. When the number of entries exceed part of
// the cache is dumped to the DB.
const postgresStateReaderTempStoreCacheSize = 1000000

// PostgresStateReaderTempStore is a postgres implementation of
// StateReaderTempStore. It's much slower than MemoryStateReaderTempStore
// but has much lower memory requirements.
type PostgresStateReaderTempStore struct {
	DSN string

	afterFirstDump bool
	// cache can contain both true and false values. false values can be set
	// when key is not in cache: then the value will be checked in a database
	// and later cached.
	cache     map[string]bool
	session   *db.Session
	tableName string
}

var cacheHit, cacheMiss int

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
	    PRIMARY KEY (key)
	) ON COMMIT DROP;`, s.tableName))
	if err != nil {
		return errors.Wrap(err, "Error creating table")
	}

	s.cache = make(map[string]bool)

	go func() {
		c := time.Tick(5 * time.Second)
		for range c {
			fmt.Println("Hit:", cacheHit, "Miss:", cacheMiss)

			cacheHit = 0
			cacheMiss = 0
		}
	}()

	return nil
}

func (s *PostgresStateReaderTempStore) Preload(keys []string) error {
	// Before first dump all `true` there are no keys in a database.
	if !s.afterFirstDump {
		return nil
	}

	start := time.Now()

	keysMap := make(map[string]bool)
	for _, key := range keys {
		// The cache has always the latest data that (maybe) was not dumped to
		// a database yet so check it first.
		_, exist := s.cache[key]
		if exist {
			continue
		}

		keysMap[key] = true
	}

	// At this point `keysMap` contains only the keys that are not currently cached.

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	rows, err := psql.Select("key").
		From(s.tableName).
		Where(map[string]interface{}{"key": keys}).
		RunWith(s.session.GetTx().Tx).
		Query()

	if err != nil {
		return err
	}

	for rows.Next() {
		var key string
		err := rows.Scan(&key)
		if err != nil {
			return err
		}

		// If value is found in a DB we cache with `true` value
		s.cache[key] = true
		delete(keysMap, key)
	}

	// For all the keys that are left (not found in a DB) we set `false` value.
	for key, _ := range keysMap {
		s.cache[key] = false
	}

	fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>> preloaded ", len(keys), " keys in: ", time.Since(start))

	return nil
}

func (s *PostgresStateReaderTempStore) Add(key string) error {
	s.cache[key] = true
	return s.dumpCacheIfNeeded()
}

func (s *PostgresStateReaderTempStore) dumpCacheIfNeeded() error {
	cacheLen := len(s.cache)
	if cacheLen < postgresStateReaderTempStoreCacheSize {
		return nil
	}

	s.afterFirstDump = true

	start := time.Now()
	query := s.newInsertBuilder()
	dumpedEntries := 0
	queryParams := 0

	for k, v := range s.cache {
		// We omit false values set in `Get`
		if !v {
			continue
		}

		query = query.Values(k)
		delete(s.cache, k)

		dumpedEntries++
		queryParams++

		// The number comes from the fact that the maximum number of params per
		// postgres query is 65535. When we are approaching the max params,
		// insert rows we have so far and create a new builder.
		if queryParams > 65000 {
			_, err := s.session.Exec(query)
			if err != nil {
				return err
			}

			query = s.newInsertBuilder()
			queryParams = 0
		}

		// Exit if 1/2 of the cache is dumped
		if dumpedEntries >= cacheLen/2 {
			break
		}
	}

	fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>> dumped ", dumpedEntries, " entries in: ", time.Since(start))

	// Insert the last batch.
	if queryParams > 0 {
		_, err := s.session.Exec(query)
		return err
	}

	return nil
}

func (s *PostgresStateReaderTempStore) newInsertBuilder() sq.InsertBuilder {
	return sq.Insert(s.tableName).
		Columns("key").
		Suffix("ON CONFLICT (key) DO NOTHING")
}

func (s *PostgresStateReaderTempStore) Exist(key string) (bool, error) {
	// Check cache first
	value, exist := s.cache[key]
	if exist {
		cacheHit++
		return value, nil
	}

	// Before first dump all `true` entries should be in cache. If it's
	// not found then return false. It improves performance a lot before
	// the first data is dumped.
	if !s.afterFirstDump {
		cacheHit++
		return false, nil
	}

	cacheMiss++

	value, err := s.dbGet(key)
	if err != nil {
		return false, err
	}

	s.cache[key] = value

	err = s.dumpCacheIfNeeded()
	return value, err
}

func (s *PostgresStateReaderTempStore) dbGet(key string) (bool, error) {
	query := sq.Select("1").
		From(s.tableName).
		Where("key = ?", key)

	var value int
	if err := s.session.Get(&value, query); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.Wrap(err, "could not get value")
	}

	// Value found in a store
	return true, nil
}

func (s *PostgresStateReaderTempStore) Close() error {
	// This will also drop temp table
	return s.session.Close()
}
