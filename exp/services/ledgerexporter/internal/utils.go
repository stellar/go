package ledgerexporter

import (
	"compress/gzip"
	"io"

	xdr3 "github.com/stellar/go-xdr/xdr3"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/storage"
)

const (
	fileSuffix = ".xdr.gz"
)

// getLatestLedgerSequenceFromHistoryArchives returns the most recent ledger sequence (checkpoint ledger)
// number present in the history archives.
func getLatestLedgerSequenceFromHistoryArchives(historyArchivesURLs []string) (uint32, error) {
	for _, historyArchiveURL := range historyArchivesURLs {
		ha, err := historyarchive.Connect(
			historyArchiveURL,
			historyarchive.ArchiveOptions{
				ConnectOptions: storage.ConnectOptions{
					UserAgent: "ledger-exporter",
				},
			},
		)
		if err != nil {
			logger.WithError(err).Warnf("Error connecting to history archive %s", historyArchiveURL)
			continue // Skip to next archive
		}

		has, err := ha.GetRootHAS()
		if err != nil {
			logger.WithError(err).Warnf("Error getting RootHAS for %s", historyArchiveURL)
			continue // Skip to next archive
		}

		return has.CurrentLedger, nil
	}

	return 0, errors.New("failed to retrieve the latest ledger sequence from any history archive")
}

type XDRGzipEncoder struct {
	XdrPayload interface{}
}

func (g *XDRGzipEncoder) WriteTo(w io.Writer) (int64, error) {
	gw := gzip.NewWriter(w)
	n, err := xdr3.Marshal(gw, g.XdrPayload)
	if err != nil {
		return int64(n), err
	}
	return int64(n), gw.Close()
}

type XDRGzipDecoder struct {
	XdrPayload interface{}
}

func (d *XDRGzipDecoder) ReadFrom(r io.Reader) (int64, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return 0, err
	}
	defer gr.Close()

	n, err := xdr3.Unmarshal(gr, d.XdrPayload)
	if err != nil {
		return int64(n), err
	}
	return int64(n), nil
}
