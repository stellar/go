package ledgerexporter

import (
	"context"
)

type ResumableManager interface {
	FindStartBoundary(ctx context.Context, start, end uint32) uint32
}

type resumableManagerService struct {
	exporterConfig ExporterConfig
	dataStore      DataStore
	networkManager NetworkManager
	network        string
}

func NewResumableManager(dataStore DataStore, exporterConfig ExporterConfig, networkManager NetworkManager, network string) ResumableManager {
	return &resumableManagerService{exporterConfig: exporterConfig, dataStore: dataStore, networkManager: networkManager, network: network}
}

// Find the nearest "LedgersPerFile" starting boundary ledger number relative to requested start which
// does not exist on datastore yet.
//
// start - start search from this ledger
// end   - stop search at this ledger.
//
// If end=0, meaning unbounded, this will substitute an effective end value of the
// most recent archived ledger number.
//
// return - the next bounded start ledger position
//
// Will be non-zero If able to identify the nearest "LedgersPerFile" starting boundary ledger number
// which is absent on datastore given start and end.
// If data store has all files up to end, then returns the next "LedgersPerFile" starting boundary ledger
// If not able to identify next boundary ledger due to any type of error, returns 0.
func (rm resumableManagerService) FindStartBoundary(ctx context.Context, start, end uint32) uint32 {
	if ctx.Err() != nil {
		return 0
	}

	// streaming mode for start, no historical point to resume from
	if start < 1 {
		return 0
	}

	// streaming mode for end, get current ledger to use for a sane bounded range during resumability check
	if end < 1 {
		var latestErr error
		end, latestErr = rm.networkManager.GetLatestLedgerSequenceFromHistoryArchives(ctx, rm.network)
		if latestErr != nil {
			logger.WithError(latestErr).Infof("Resumability of requested export ledger range start=%d, end=%d, was not able to get latest ledger from network %v", start, end, rm.network)
			return 0
		}
		if start > end {
			// requested to start at a point beyond the latest network, resume not applicable.
			return 0
		}
	}

	binarySearchStart := start
	binarySearchStop := end
	nearestAbsentLedger := uint32(0)
	lookupCache := map[string]bool{}

	for binarySearchStart <= binarySearchStop {
		if ctx.Err() != nil {
			return 0
		}

		binarySearchMiddle := (binarySearchStop-binarySearchStart)/2 + binarySearchStart
		objectKeyMiddle := rm.exporterConfig.GetObjectKeyFromSequenceNumber(binarySearchMiddle)

		// there may be small occurrence of repeated queries on same object key once
		// search narrows down to a range that fits within the ledgers per file
		// worst case being 'log of ledgers_per_file' queries.
		middleFoundOnStore, foundInCache := lookupCache[objectKeyMiddle]
		if !foundInCache {
			var datastoreErr error
			middleFoundOnStore, datastoreErr = rm.dataStore.Exists(ctx, objectKeyMiddle)
			if datastoreErr != nil {
				logger.WithError(datastoreErr).Infof("For resuming of export ledger range start=%d, end=%d, was not able to check if objec key %v exists on data store", start, end, objectKeyMiddle)
				return 0
			}
			lookupCache[objectKeyMiddle] = middleFoundOnStore
		}

		if middleFoundOnStore {
			binarySearchStart = binarySearchMiddle + 1
		} else {
			nearestAbsentLedger = binarySearchMiddle
			binarySearchStop = binarySearchMiddle - 1
		}
	}

	if nearestAbsentLedger > 0 {
		// return the boundary start for the ledger that search confirmed missing on data store
		return rm.exporterConfig.GetSequenceNumberStartBoundary(nearestAbsentLedger)
	}

	// data store had all ledgers for requested range, return the next boundary start
	return rm.exporterConfig.GetSequenceNumberEndBoundary(end) + 1
}
