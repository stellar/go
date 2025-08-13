package galexie

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest/ledgerbackend"
	"github.com/stellar/go/support/compressxdr"
	"github.com/stellar/go/support/datastore"
	"github.com/stellar/go/support/galexie"
	supporthttp "github.com/stellar/go/support/http"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/ordered"
)

const (
	adminServerReadTimeout     = 5 * time.Second
	adminServerShutdownTimeout = time.Second * 5
	// TODO: make this timeout configurable
	uploadShutdownTimeout = 10 * time.Second
	// We expect the queue size to rarely exceed 1 or 2 because
	// upload speeds are expected to be much faster than the rate at which
	// captive core emits ledgers. However, configuring a higher capacity
	// than our expectation is useful because if we observe a large queue
	// size in our metrics that is an indication that uploads to the
	// data store have degraded
	uploadQueueCapacity = 128
	nameSpace           = "galexie"
)

var (
	logger  = log.New().WithField("service", nameSpace)
	version = "develop"
)

func NewDataAlreadyExportedError(Start uint32, End uint32) *DataAlreadyExportedError {
	return &DataAlreadyExportedError{
		Start: Start,
		End:   End,
	}
}

type DataAlreadyExportedError struct {
	Start uint32
	End   uint32
}

func (m DataAlreadyExportedError) Error() string {
	return fmt.Sprintf("For export ledger range start=%d, end=%d, the remote storage has all the data, there is no need to continue export", m.Start, m.End)
}

func NewInvalidDataStoreError(LedgerSequence uint32, LedgersPerFile uint32) *InvalidDataStoreError {
	return &InvalidDataStoreError{
		LedgerSequence: LedgerSequence,
		LedgersPerFile: LedgersPerFile,
	}
}

type InvalidDataStoreError struct {
	LedgerSequence uint32
	LedgersPerFile uint32
}

func (m InvalidDataStoreError) Error() string {
	return fmt.Sprintf("The remote data store has inconsistent data, "+
		"a resumable starting ledger of %v was identified, "+
		"but that is not aligned to expected ledgers-per-file of %v. use 'scan-and-fill' sub-command to bypass",
		m.LedgerSequence, m.LedgersPerFile)
}

type App struct {
	config        *Config
	ledgerBackend ledgerbackend.LedgerBackend
	dataStore     datastore.DataStore
	exportManager *ExportManager
	manifest      galexie.Manifest
	uploader      Uploader
	adminServer   *http.Server
}

func NewApp() *App {
	logger.SetLevel(log.DebugLevel)
	app := &App{}
	return app
}

func (a *App) init(ctx context.Context, runtimeSettings RuntimeSettings) error {
	var err error
	var archive historyarchive.ArchiveInterface

	logger.Infof("Starting Galexie with version %s", version)

	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{Namespace: nameSpace}),
		collectors.NewGoCollector(),
	)

	if a.config, err = NewConfig(runtimeSettings, nil); err != nil {
		return errors.Wrap(err, "Could not load configuration")
	}
	if archive, err = a.config.GenerateHistoryArchive(ctx, logger); err != nil {
		return err
	}
	if err = a.config.ValidateLedgerRange(ctx, archive); err != nil {
		return err
	}

	if a.dataStore, err = datastore.NewDataStore(ctx, a.config.DataStoreConfig); err != nil {
		return fmt.Errorf("could not connect to destination data store %w", err)
	}

	if err = validateExistingFileExtension(ctx, a.dataStore); err != nil {
		return err
	}

	logger.Infof("Attempting to configure datastore...")
	var created bool
	a.manifest, created, err = a.createManifestIfNotExists(ctx)
	if err != nil {
		return fmt.Errorf("could not configure datastore %w", err)
	}

	if created {
		logger.WithField("manifest", a.manifest).Infof("Successfully created datastore config manifest.")
	} else {
		logger.WithField("manifest", a.manifest).Infof("Datastore config manifest already exists.")
	}
	schema := galexie.Schema{
		LedgersPerFile:    a.manifest.LedgersPerBatch,
		FilesPerPartition: a.manifest.BatchesPerPartition,
	}

	adjustLedgerRange(a.config, schema)

	if a.config.Resumable() {
		if err = a.applyResumability(ctx, schema, galexie.NewResumableManager(a.dataStore, schema, archive)); err != nil {
			return err
		}
	}

	logger.Infof("Final computed ledger range for backend retrieval and export, start=%d, end=%d",
		a.config.StartLedger, a.config.EndLedger)

	if a.ledgerBackend, err = newLedgerBackend(a.config, registry); err != nil {
		return err
	}

	queue := NewUploadQueue(uploadQueueCapacity, registry)
	if a.exportManager, err = NewExportManager(schema,
		a.ledgerBackend, queue, registry,
		a.config.StellarCoreConfig.NetworkPassphrase,
		a.config.CoreVersion); err != nil {
		return err
	}
	a.uploader = NewUploader(a.dataStore, queue, registry)

	if a.config.AdminPort != 0 {
		a.adminServer = newAdminServer(a.config.AdminPort, registry)
	}
	return nil
}

func validateExistingFileExtension(ctx context.Context, ds datastore.DataStore) error {
	fileExt, err := galexie.GetLedgerFileExtension(ctx, ds)
	if err != nil {
		if errors.Is(err, galexie.ErrNoLedgerFiles) {
			// Empty data lake is OK. will bootstrap with .zst going forward.
			log.Infof("no existing ledger files found in data store")
			return nil
		}
		return fmt.Errorf("unable to determine ledger file extension from data store: %w", err)
	}

	if fileExt != compressxdr.DefaultCompressor.Name() {
		return fmt.Errorf("detected older incompatible ledger files in the data store (extension %q). "+
			"Galexie v23.0+ requires starting with an empty datastore", fileExt)
	}

	return nil
}

func (a *App) applyResumability(ctx context.Context, schema galexie.Schema, resumableManager galexie.ResumableManager) error {
	absentLedger, ok, err := resumableManager.FindStart(ctx, a.config.StartLedger, a.config.EndLedger)
	if err != nil {
		return err
	}
	if !ok {
		return NewDataAlreadyExportedError(a.config.StartLedger, a.config.EndLedger)
	}

	// TODO - evaluate a more robust validation of remote data for ledgers-per-file consistency
	// this assumes ValidateLedgerRange() has conditioned the a.config.StartLedger to be at least > 1
	if absentLedger > 2 && absentLedger != schema.GetSequenceNumberStartBoundary(absentLedger) {
		return NewInvalidDataStoreError(absentLedger, schema.LedgersPerFile)
	}
	logger.Infof("For export ledger range start=%d, end=%d, the remote storage has some of this data already, will resume at later start ledger of %d", a.config.StartLedger, a.config.EndLedger, absentLedger)
	a.config.StartLedger = absentLedger

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

func newAdminServer(adminPort int, prometheusRegistry *prometheus.Registry) *http.Server {
	mux := supporthttp.NewMux(logger)
	mux.Handle("/metrics", promhttp.HandlerFor(prometheusRegistry, promhttp.HandlerOpts{}))
	adminAddr := fmt.Sprintf(":%d", adminPort)
	return &http.Server{
		Addr:        adminAddr,
		Handler:     mux,
		ReadTimeout: adminServerReadTimeout,
	}
}

func (a *App) Run(runtimeSettings RuntimeSettings) error {
	ctx, cancel := context.WithCancel(runtimeSettings.Ctx)
	defer cancel()

	if err := a.init(ctx, runtimeSettings); err != nil {
		var dataAlreadyExported *DataAlreadyExportedError
		if errors.As(err, &dataAlreadyExported) {
			logger.Info(err.Error())
			logger.Info("Shutting down Galexie")
			return nil
		}
		logger.WithError(err).Error("Stopping Galexie")
		return err
	}
	defer a.close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		err := a.uploader.Run(ctx, uploadShutdownTimeout)
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

	if a.adminServer != nil {
		// no need to include this goroutine in the wait group
		// because a.adminServer.Shutdown() is called below and
		// that will block until a.adminServer has finished
		// shutting down
		go func() {
			logger.Infof("Starting admin server on port %v", a.config.AdminPort)
			if err := a.adminServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Warn(errors.Wrap(err, "error in internalServer.ListenAndServe()"))
			}
		}()
	}

	// Handle OS signals to gracefully terminate the service
	sigCh := make(chan os.Signal, 1)
	defer close(sigCh)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig, ok := <-sigCh
		if ok {
			logger.Infof("Received termination signal: %v", sig)
			cancel()
		}
	}()

	wg.Wait()
	logger.Info("Shutting down Galexie")

	if a.adminServer != nil {
		serverShutdownCtx, serverShutdownCancel := context.WithTimeout(context.Background(), adminServerShutdownTimeout)
		defer serverShutdownCancel()

		if err := a.adminServer.Shutdown(serverShutdownCtx); err != nil {
			logger.WithError(err).Warn("error in internalServer.Shutdown")
		}
	}
	return nil
}

func (a *App) createManifestIfNotExists(ctx context.Context) (galexie.Manifest, bool, error) {
	manifest, err := galexie.GetManifest(ctx, a.dataStore)
	if err == nil {
		// Validate that the existing manifest matches the provided config
		if manifest.NetworkPassphrase != a.config.StellarCoreConfig.NetworkPassphrase {
			return galexie.Manifest{}, false, fmt.Errorf(
				"network passphrase in manifest (%s) does not match configured network passphrase (%s)",
				a.config.StellarCoreConfig.NetworkPassphrase,
				manifest.NetworkPassphrase,
			)
		}
		if manifest.Compression != compressionType {
			return galexie.Manifest{}, false, fmt.Errorf(
				"compression type in manifest (%s) does not match configured compression type (%s)",
				manifest.Compression,
				compressionType,
			)
		}
		if a.config.Schema != nil {
			if manifest.BatchesPerPartition != a.config.Schema.FilesPerPartition {
				return galexie.Manifest{}, false, fmt.Errorf(
					"files per partition in manifest (%d) does not match configured files per partition (%d)",
					manifest.BatchesPerPartition,
					a.config.Schema.FilesPerPartition,
				)
			}
			if manifest.LedgersPerBatch != a.config.Schema.LedgersPerFile {
				return galexie.Manifest{}, false, fmt.Errorf(
					"ledgers per file in manifest (%d) does not match configured ledgers per file (%d)",
					manifest.LedgersPerBatch,
					a.config.Schema.LedgersPerFile,
				)
			}
		}
		return manifest, false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return galexie.Manifest{}, false, fmt.Errorf("failed to read manifest: %w", err)
	}

	if a.config.Schema == nil {
		return galexie.Manifest{}, false, fmt.Errorf("no schema provided, cannot create manifest")
	}

	manifest = galexie.Manifest{
		NetworkPassphrase:   a.config.StellarCoreConfig.NetworkPassphrase,
		Version:             galexie.ManifestVersion,
		Compression:         compressionType,
		LedgersPerBatch:     a.config.Schema.LedgersPerFile,
		BatchesPerPartition: a.config.Schema.FilesPerPartition,
	}
	return manifest, true, galexie.WriteManifest(ctx, a.dataStore, manifest)
}

func adjustLedgerRange(config *Config, schema galexie.Schema) {
	// Check if either the start or end ledger does not fall on the "LedgersPerFile" boundary
	// and adjust the start and end ledger accordingly.
	// Align the start ledger to the nearest "LedgersPerFile" boundary.
	config.StartLedger = schema.GetSequenceNumberStartBoundary(config.StartLedger)

	// Ensure that the adjusted start ledger is at least 2.
	config.StartLedger = ordered.Max(2, config.StartLedger)

	// Align the end ledger (for bounded cases) to the nearest "LedgersPerFile" boundary.
	if config.EndLedger != 0 {
		config.EndLedger = schema.GetSequenceNumberEndBoundary(config.EndLedger)
	}

	logger.Infof("Computed effective export boundary ledger range: start=%d, end=%d", config.StartLedger, config.EndLedger)
}

// newLedgerBackend Creates and initializes captive core ledger backend
// Currently, only supports captive-core as ledger backend
func newLedgerBackend(config *Config, prometheusRegistry *prometheus.Registry) (ledgerbackend.LedgerBackend, error) {
	// best effort check on a core bin available from PATH to provide as default if
	// no core bin is provided from config.
	coreBinFromPath, _ := exec.LookPath("stellar-core")
	captiveConfig, err := config.GenerateCaptiveCoreConfig(coreBinFromPath)
	if err != nil {
		return nil, err
	}

	var backend ledgerbackend.LedgerBackend
	// Create a new captive core backend
	backend, err = ledgerbackend.NewCaptive(captiveConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create captive-core instance")
	}
	backend = ledgerbackend.WithMetrics(backend, prometheusRegistry, nameSpace)

	return backend, nil
}
