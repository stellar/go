// Copyright 2021 Stellar Development Foundation and contributors. Licensed
// under the Apache License, Version 2.0. See the COPYING file at the root
// of this distribution or at http://www.apache.org/licenses/LICENSE-2.0

package historyarchive

import (
	"math/rand"

	"github.com/stellar/go/support/errors"
)

// A PooledArchive is just a collection of `ArchiveInterface`s so that we can
// distribute requests fairly throughout the pool.
type PooledArchive []ArchiveInterface

// CreatePool tries connecting to each of the provided history archive URLs,
// returning a pool of valid archives.
//
// If none of the archives work, this returns the error message of the last
// failed archive. Note that the errors for each individual archive are hard to
// track if there's success overall.
//
// Possible FIXME for the above limitation: return []error instead? but then
// users need to check `len(pool) > 0` instead of `err == nil`.
func CreatePool(archiveURLs []string, config ConnectOptions) (*PooledArchive, error) {
	if len(archiveURLs) <= 0 {
		return nil, errors.New("No history archives provided")
	}

	var lastErr error = nil

	// Try connecting to all of the listed archives, but only store valid ones.
	var validArchives PooledArchive
	for _, url := range archiveURLs {
		archive, err := Connect(
			url,
			ConnectOptions{
				NetworkPassphrase:   config.NetworkPassphrase,
				CheckpointFrequency: config.CheckpointFrequency,
				Context:             config.Context,
			},
		)

		if err != nil {
			lastErr = errors.Wrapf(err, "Error connecting to history archive (%s)", url)
			continue
		}

		validArchives = append(validArchives, archive)
	}

	if len(validArchives) == 0 {
		return nil, lastErr
	}

	return &validArchives, nil
}

func GetRandomArchive(pool PooledArchive) ArchiveInterface {
	return pool[rand.Intn(len(pool))]
}
