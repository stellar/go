package ledgerexporter

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"github.com/stellar/go/ingest/ledgerbackend"
	_ "github.com/stellar/go/network"
	"github.com/stellar/go/support/log"
)

var (
	logger = log.New().WithField("service", "ledger-exporter")
)

type DataAlreadyExported error

func NewDataAlreadyExported(Start uint32, End uint32) DataAlreadyExported {
	return &dataAlreadyExported{
		Start: Start,
		End:   End,
	}
}

type dataAlreadyExported struct {
	Start uint32
	End   uint32
}

func (m dataAlreadyExported) Error() string {
	return fmt.Sprintf("For export ledger range start=%d, end=%d, the remote storage has all the data, there is no need to continue export", m.Start, m.End)
}

type App struct {
	config        Config
	ledgerBackend ledgerbackend.LedgerBackend
	dataStore     DataStore
	exportManager ExportManager
	uploader      Uploader
	flags         Flags
}

func NewApp(flags Flags) *App {
	logger.SetLevel(log.DebugLevel)
	app := &App{flags: flags}
	return app
}

func (a *App) init(ctx context.Context) error {
	var config *Config
	var err error

	if config, err = NewConfig(ctx, NetworkManagerService, a.flags); err != nil {
		return errors.Wrap(err, "Could not load configuration")
	}
	a.config = *config

	if a.dataStore, err = NewDataStore(ctx, a.config.DataStoreConfig, a.config.Network, a.config.LedgerBatchConfig); err != nil {
		return errors.Wrap(err, "Could not connect to destination data store")
	}

	resumableManager := NewResumableManager(a.dataStore, a.config.LedgerBatchConfig, NetworkManagerService, a.config.Network)
	resumableStartLedger, dataStoreComplete := resumableManager.FindStartBoundary(ctx, a.config.StartLedger, a.config.EndLedger)
	if dataStoreComplete {
		return NewDataAlreadyExported(a.config.StartLedger, a.config.EndLedger)
	}

	if resumableStartLedger > 0 {
		// resumable is a best effort attempt, if response is 0 that means no resume point was obtainable.
		logger.Infof("For export ledger range start=%d, end=%d, the remote storage has some of this data already, will resume at later start ledger of %d", config.StartLedger, config.EndLedger, resumableStartLedger)
		a.config.StartLedger = resumableStartLedger
	}

	logger.Infof("Final computed ledger range for backend retrieval and export, start=%d, end=%d", a.config.StartLedger, a.config.EndLedger)

	if a.ledgerBackend, err = newLedgerBackend(ctx, a.config); err != nil {
		return err
	}
	if a.exportManager, err = NewExportManager(a.config.LedgerBatchConfig, a.ledgerBackend); err != nil {
		return err
	}
	a.uploader = NewUploader(a.dataStore, a.exportManager.GetMetaArchiveChannel())

	return nil
}

func (a *App) close() {
	if err := a.dataStore.Close(); err != nil {
		logger.WithError(err).Error("Error closing datastore")
	}
	if err := a.ledgerBackend.Close(); err != nil {
		logger.WithError(err).Error("Error closing ledgerBackend")
	}
}

func (a *App) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := a.init(ctx); err != nil {
		var dataAlreadyExported *DataAlreadyExported
		if errors.As(err, dataAlreadyExported) {
			logger.Info(err.Error())
			logger.Info("Shutting down ledger-exporter")
			return
		}
		logger.WithError(err).Fatal("Stopping ledger-exporter")
	}
	defer a.close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		err := a.uploader.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.WithError(err).Error("Error executing Uploader")
			cancel()
		}
	}()

	go func() {
		defer wg.Done()

		err := a.exportManager.Run(ctx, a.config.StartLedger, a.config.EndLedger)
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.WithError(err).Error("Error executing ExportManager")
			cancel()
		}
	}()

	// Handle OS signals to gracefully terminate the service
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Infof("Received termination signal: %v", sig)
		cancel()
	}()

	wg.Wait()
	logger.Info("Shutting down ledger-exporter")
}

// newLedgerBackend Creates and initializes captive core ledger backend
// Currently, only supports captive-core as ledger backend
func newLedgerBackend(ctx context.Context, config Config) (ledgerbackend.LedgerBackend, error) {
	captiveConfig, err := config.GenerateCaptiveCoreConfig()
	if err != nil {
		return nil, err
	}

	// Create a new captive core backend
	backend, err := ledgerbackend.NewCaptive(captiveConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create captive-core instance")
	}

	var ledgerRange ledgerbackend.Range
	if config.EndLedger == 0 {
		ledgerRange = ledgerbackend.UnboundedRange(config.StartLedger)
	} else {
		ledgerRange = ledgerbackend.BoundedRange(config.StartLedger, config.EndLedger)
	}

	if err = backend.PrepareRange(ctx, ledgerRange); err != nil {
		return nil, errors.Wrap(err, "Could not prepare captive core ledger backend")
	}
	return backend, nil
}
