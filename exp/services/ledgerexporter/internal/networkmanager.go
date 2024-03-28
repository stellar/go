package ledgerexporter

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/storage"
)

var (
	NetworkManagerService = &networkManagerService{}
)

type NetworkManager interface {
	GetLatestLedgerSequenceFromHistoryArchives(ctx context.Context, networkName string) (uint32, error)
}

type networkManagerService struct{}

// getLatestLedgerSequenceFromHistoryArchives returns the most recent ledger sequence (checkpoint ledger)
// number present in the history archives.
func (sn networkManagerService) GetLatestLedgerSequenceFromHistoryArchives(ctx context.Context, networkName string) (uint32, error) {
	var historyArchiveUrls []string
	switch networkName {
	case Pubnet:
		historyArchiveUrls = network.PublicNetworkhistoryArchiveURLs
	case Testnet:
		historyArchiveUrls = network.TestNetworkhistoryArchiveURLs
	default:
		return 0, errors.Errorf("Invalid network %s", networkName)
	}

	archivePool, err := historyarchive.NewArchivePool(historyArchiveUrls, historyarchive.ArchiveOptions{
		ConnectOptions: storage.ConnectOptions{
			UserAgent: "ledger-exporter",
			Context:   ctx,
		},
	})

	if err != nil {
		logger.WithError(err).Warnf("Error creating history archive pool, %v", historyArchiveUrls)
		return 0, err
	}

	has, err := archivePool.GetRootHAS()
	if err != nil {
		logger.WithError(err).Warnf("Error getting root HAS from archives, %v", historyArchiveUrls)
		return 0, errors.Wrap(err, "failed to retrieve the latest ledger sequence from any history archive")
	}

	return has.CurrentLedger, nil
}
