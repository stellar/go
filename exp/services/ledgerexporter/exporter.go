package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/storage"
	"github.com/stellar/go/xdr"
	"io"
	"strconv"
)

// ExportObject represents a file with metadata and binary data.
type ExportObject struct {
	// metadata
	startLedger   uint32
	endLedger     uint32
	objectName    string
	partitionName string
	// Actual binary data
	ledgerTxMetas xdr.TxMetaLedgerBatch
}

func NewExportObject(ledgerSeq uint32) *ExportObject {
	exportObject := &ExportObject{
		startLedger: ledgerSeq,
		endLedger:   ledgerSeq,
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
	fileMap        map[string]*ExportObject
	exportObjectCh chan *ExportObject
}

// NewExporter creates a new Exporter with the provided configuration.
func NewExporter(config ExporterConfig, store storage.Storage, backend ledgerbackend.LedgerBackend) *Exporter {
	ex := Exporter{
		config:         config,
		destination:    store,
		backend:        backend,
		fileMap:        make(map[string]*ExportObject),
		exportObjectCh: make(chan *ExportObject),
	}
	ex.StartUploader()
	return &ex
}

// AddLedgerTxMeta adds ledger metadata to the current export object.
func (e *Exporter) AddLedgerTxMeta(txMeta xdr.LedgerCloseMeta) error {
	//e.mutex.Lock()
	//defer e.mutex.Unlock()

	// determine filename for the given ledger sequence
	objectName := e.getObjectName(txMeta.LedgerSequence())
	exportObject, exists := e.fileMap[objectName]

	if !exists {
		// Create a new ExportObject and add it to the map
		exportObject = NewExportObject(txMeta.LedgerSequence())
		exportObject.objectName = objectName
		exportObject.partitionName = e.getPartitionName(txMeta.LedgerSequence())
		e.fileMap[objectName] = exportObject
	}

	exportObject.AddLedgerTxMeta(txMeta)

	if exportObject.NumberOfLedgers() == e.config.LedgersPerFile {
		// Current export object is full, send it for upload
		e.exportObjectCh <- exportObject
	}
	return nil
}

// Upload uploads the serialized binary data of ledger TxMeta
// to the specified destination
func (e *Exporter) upload(object *ExportObject) error {
	logger.Infof("Uploading: %s", object.partitionName+"/"+object.objectName)
	blob, err := object.ledgerTxMetas.MarshalBinary()
	if err != nil {
		return err
	}

	return e.destination.PutFile(
		object.partitionName+"/"+object.objectName,
		io.NopCloser(bytes.NewReader(blob)),
	)
}

// StartUploader starts the uploader goroutine
func (e *Exporter) StartUploader() {
	go func() {
		for {
			// Receive ExportObject from the channel
			exportObject := <-e.exportObjectCh

			// Upload the ExportObject
			err := e.upload(exportObject)
			if err != nil {
				// Handle error if needed
				fmt.Println("Error uploading:", err)
			}
			delete(e.fileMap, exportObject.objectName)
		}
	}()
}

// getPartitionName generates the directory prefix based on the ledger sequence.
func (e *Exporter) getPartitionName(ledgerSeq uint32) string {
	partitionSize := e.config.LedgersPerFile * e.config.FilesPerPartition
	start := (ledgerSeq / partitionSize) * partitionSize
	end := start + partitionSize - 1

	return strconv.Itoa(int(start)) + "-" + strconv.Itoa(int(end))
}

// getObjectName generates the file name based on the ledger sequence.
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
			//time.Sleep(time.Duration(1) * time.Second)
			e.AddLedgerTxMeta(ledger)
			nextLedger++
		}
	}
}
