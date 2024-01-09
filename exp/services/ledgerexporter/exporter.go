package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/storage"
	"io"
	"strconv"
	"time"

	"github.com/stellar/go/xdr"
)

// ExportObject represents a file with metadata and binary data.
type ExportObject struct {
	// metadata
	startLedger uint32
	endLedger   uint32
	fullPath    string
	// Actual binary data
	ledgerTxMetas xdr.TxMetaLedgerBatch
}

func NewExportObject(ledgerSeq uint32, fullPath string) *ExportObject {
	exportObject := &ExportObject{
		startLedger: ledgerSeq,
		endLedger:   ledgerSeq,
		fullPath:    fullPath,
	}
	exportObject.ledgerTxMetas.StartSequence = xdr.Uint32(ledgerSeq)
	return exportObject
}

// AddLedgerTxMeta adds a TxMeta for a ledger to LedgerTxMetas
func (e *ExportObject) AddLedgerTxMeta(txMeta xdr.LedgerCloseMeta) {
	e.ledgerTxMetas.TxMetas = append(e.ledgerTxMetas.TxMetas, txMeta)
	e.ledgerTxMetas.EndSequence = xdr.Uint32(txMeta.LedgerSequence())
	e.endLedger = txMeta.LedgerSequence()
}

func (e *ExportObject) NumberOfLedgers() uint32 {
	return uint32(len(e.ledgerTxMetas.TxMetas))
}

type ExporterConfig struct {
	LedgersPerFile    uint32 `toml:"ledgers_per_file"`
	FilesPerPartition uint32 `toml:"files_per_partition"`
	DestinationUrl    string `toml:"destination_url"`
}

// Exporter manages the creation and handling of export objects.
type Exporter struct {
	config      ExporterConfig
	destination storage.Storage
	backend     ledgerbackend.LedgerBackend
	//	mutex       sync.Mutex
	fileMap map[string]*ExportObject
}

// NewExporter creates a new Exporter with the provided configuration.
func NewExporter(config ExporterConfig, store storage.Storage, backend ledgerbackend.LedgerBackend) *Exporter {
	return &Exporter{
		config:      config,
		destination: store,
		backend:     backend,
		fileMap:     make(map[string]*ExportObject),
	}
}

// AddLedgerTxMeta adds ledger metadata to the current export object.
func (e *Exporter) AddLedgerTxMeta(txMeta xdr.LedgerCloseMeta) error {
	//e.mutex.Lock()
	//defer e.mutex.Unlock()

	objectName := e.getObjectName(txMeta.LedgerSequence())
	exportObject, exists := e.fileMap[objectName]
	if !exists {
		fullPath := e.getPartitionName(txMeta.LedgerSequence()) + "/" + objectName
		exportObject = NewExportObject(txMeta.LedgerSequence(), fullPath)
		e.fileMap[objectName] = exportObject
	}
	exportObject.AddLedgerTxMeta(txMeta)
	if exportObject.NumberOfLedgers() == e.config.LedgersPerFile {
		// Current export object is full, upload it
		e.Upload(exportObject)
		fmt.Printf("seq:%d %s\n", txMeta.LedgerSequence(), exportObject.fullPath)
		delete(e.fileMap, objectName)
		exportObject = nil
	}

	return nil
}

// Upload uploads the serialized binary data of ledger TxMeta
// to the specified destination
func (e *Exporter) Upload(object *ExportObject) error {
	blob, err := object.ledgerTxMetas.MarshalBinary()
	if err != nil {
		return err
	}
	return e.destination.PutFile(
		object.fullPath,
		io.NopCloser(bytes.NewReader(blob)),
	)
}

// getPartitionName generates the directory prefix based on the ledger sequence.
func (e *Exporter) getPartitionName(ledgerSeq uint32) string {
	partitionSize := e.config.LedgersPerFile * e.config.FilesPerPartition
	start := (ledgerSeq / partitionSize) * partitionSize
	end := start + partitionSize - 1

	return strconv.Itoa(int(start)) + "-" + strconv.Itoa(int(end))
}

// getFileName generates the file name based on the ledger sequence.
func (e *Exporter) getObjectName(ledgerSeq uint32) string {
	start := (ledgerSeq / e.config.LedgersPerFile) * e.config.LedgersPerFile
	end := start + e.config.LedgersPerFile - 1

	return strconv.Itoa(int(start)) + "-" + strconv.Itoa(int(end))
}

// Run iterates over the specified range of ledger numbers and uploads
// the corresponding serialized binary data to the destination. The process
// continues until the ending ledger number is reached.
func (e *Exporter) Run(ctx context.Context, startLedger, endLedger uint32) {

	for nextLedger := startLedger; endLedger < 1 || nextLedger <= endLedger; {
		select {
		case <-ctx.Done():
			logger.Info("Exporter stopped due to context cancellation.")
			return
		default:
			ledger, err := e.backend.GetLedger(ctx, nextLedger)
			if err != nil {
				//Handle error
			}
			time.Sleep(time.Duration(1) * time.Second)
			e.AddLedgerTxMeta(ledger)
			nextLedger++
		}
	}
}
