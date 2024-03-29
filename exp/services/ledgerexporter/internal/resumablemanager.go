package ledgerexporter

import (
	"context"
)

type ResumableManager interface {
	FindStartBoundary(ctx context.Context, start, end uint32) (resumableLedger uint32, dataStoreComplete bool)
}

type resumableManagerService struct {
	exporterConfig LedgerBatchConfig
	dataStore      DataStore
	networkManager NetworkManager
	network        string
}

func NewResumableManager(dataStore DataStore, exporterConfig LedgerBatchConfig, networkManager NetworkManager, network string) ResumableManager {
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
// return:
// resumableLedger - if > 0, will be the next ledger that is not populated on data store.
// dataStoreComplete - if true, there was no gaps on data store for bounded range requested
//
// if resumableLedger is 0 and dataStoreComplete is false, no resumability was possible.
func (rm resumableManagerService) FindStartBoundary(ctx context.Context, start, end uint32) (resumableLedger uint32, dataStoreComplete bool) {
	// streaming mode for start, no historical point to resume from
	if start < 1 {
		return 0, false
	}

	// streaming mode for end, get current ledger to use for a sane bounded range during resumability check
	networkLatest := uint32(0)
	if end < 1 {
		var latestErr error
		networkLatest, latestErr = rm.networkManager.GetLatestLedgerSequenceFromHistoryArchives(ctx, rm.network)
		if latestErr != nil {
			logger.WithError(latestErr).Infof("Resumability of requested export ledger range start=%d, end=%d, was not able to get latest ledger from network %v", start, end, rm.network)
			return 0, false
		}
		logger.Infof("Resumability resovled unbounded to latest ledger =%d for network=%v", networkLatest, rm.network)

		if start > networkLatest {
			// requested to start at a point beyond the latest network, resume not applicable.
			return 0, false
		}
	}

	binarySearchStop := end
	if networkLatest > 0 {
		binarySearchStop = networkLatest
	}
	binarySearchStart := start
	nearestAbsentLedger := uint32(0)
	lookupCache := map[string]bool{}

	logger.Infof("Resumability searching datastore for next absent object key between ledgers %d and %d", start, end)

	for binarySearchStart <= binarySearchStop {
		if ctx.Err() != nil {
			return 0, false
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
				logger.WithError(datastoreErr).Infof("While searching datastore for resumability within export ledger range start=%d, end=%d, was not able to check if object key %v exists on data store", start, end, objectKeyMiddle)
				return 0, false
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

	//
	if nearestAbsentLedger > 0 {
		nearestAbsentBoundaryLedger := rm.exporterConfig.GetSequenceNumberStartBoundary(nearestAbsentLedger)
		logger.Infof("Resumability found next absent object start key of %d between ledgers %d and %d", nearestAbsentBoundaryLedger, start, end)
		return nearestAbsentBoundaryLedger, false
	}

	// unbounded, and datastore had up to latest network, return the start for youngest ledger on data store
	if networkLatest > 0 {
		return rm.exporterConfig.GetSequenceNumberStartBoundary(networkLatest), false
	}

	// data store had all ledgers for requested range, no resumability needed.
	logger.Infof("Resumability found no absent object start keys between ledgers %d and %d", start, end)
	return 0, true
}
