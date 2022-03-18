package ledgerbackend

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stellar/go/support/log"
)

type stellarCoreRunnerInterface interface {
	catchup(from, to uint32) error
	runFrom(from uint32, hash string) error
	getMetaPipe() <-chan metaResult
	context() context.Context
	getProcessExitError() (bool, error)
	close() error
}

type stellarCoreRunnerMode int

const (
	stellarCoreRunnerModeOnline stellarCoreRunnerMode = iota
	stellarCoreRunnerModeOffline
)

// stellarCoreRunner uses a named pipe ( https://en.wikipedia.org/wiki/Named_pipe ) to stream ledgers directly
// from Stellar Core
type pipe struct {
	// stellarCoreRunner will be reading ledgers emitted by Stellar Core from the pipe.
	// After the Stellar Core process exits, stellarCoreRunner should eventually close the reader.
	Reader io.ReadCloser
	// stellarCoreRunner is responsible for closing the named pipe file after the Stellar Core process exits.
	// However, only the Stellar Core process will be writing to the pipe. stellarCoreRunner should not
	// write anything to the named pipe file which is why the type of File is io.Closer.
	File io.Closer
}

type stellarCoreRunner struct {
	executablePath string

	started      bool
	cmd          *exec.Cmd
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	ledgerBuffer *bufferedLedgerMetaReader
	pipe         pipe
	mode         stellarCoreRunnerMode

	lock             sync.Mutex
	processExited    bool
	processExitError error

	storagePath string
	useDB       bool
	nonce       string

	log *log.Entry
}

func createRandomHexString(n int) string {
	hex := []rune("abcdef1234567890")
	b := make([]rune, n)
	for i := range b {
		b[i] = hex[rand.Intn(len(hex))]
	}
	return string(b)
}

func newStellarCoreRunner(config CaptiveCoreConfig, mode stellarCoreRunnerMode) (*stellarCoreRunner, error) {
	var fullStoragePath string
	if runtime.GOOS == "windows" || mode == stellarCoreRunnerModeOffline {
		// On Windows, first we ALWAYS append something to the base storage path,
		// because we will delete the directory entirely when Horizon stops. We also
		// add a random suffix in order to ensure that there aren't naming
		// conflicts.
		// This is done because it's impossible to send SIGINT on Windows so
		// buckets can become corrupted.
		// We also want to use random directories in offline mode (reingestion)
		// because it's possible it's running multiple Stellar-Cores on a single
		// machine.
		fullStoragePath = path.Join(config.StoragePath, "captive-core-"+createRandomHexString(8))
	} else {
		// Use the specified directory to store Captive Core's data:
		//    https://github.com/stellar/go/issues/3437
		// but be sure to re-use rather than replace it:
		//    https://github.com/stellar/go/issues/3631
		fullStoragePath = path.Join(config.StoragePath, "captive-core")
	}

	info, err := os.Stat(fullStoragePath)
	if os.IsNotExist(err) {
		innerErr := os.MkdirAll(fullStoragePath, os.FileMode(int(0755))) // rwx|rx|rx
		if innerErr != nil {
			return nil, errors.Wrap(innerErr, fmt.Sprintf(
				"failed to create storage directory (%s)", fullStoragePath))
		}
	} else if !info.IsDir() {
		return nil, errors.New(fmt.Sprintf("%s is not a directory", fullStoragePath))
	} else if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf(
			"error accessing storage directory (%s)", fullStoragePath))
	}

	ctx, cancel := context.WithCancel(config.Context)

	runner := &stellarCoreRunner{
		executablePath: config.BinaryPath,
		ctx:            ctx,
		cancel:         cancel,
		storagePath:    fullStoragePath,
		useDB:          config.UseDB,
		mode:           mode,
		nonce: fmt.Sprintf(
			"captive-stellar-core-%x",
			rand.New(rand.NewSource(time.Now().UnixNano())).Uint64(),
		),
		log: config.Log,
	}

	if conf, err := writeConf(config.Toml, mode, runner.getConfFileName()); err != nil {
		return nil, errors.Wrap(err, "error writing configuration")
	} else {
		runner.log.Debugf("captive core config file contents:\n%s", conf)
	}

	return runner, nil
}

func writeConf(captiveCoreToml *CaptiveCoreToml, mode stellarCoreRunnerMode, location string) (string, error) {
	text, err := generateConfig(captiveCoreToml, mode)
	if err != nil {
		return "", err
	}

	return string(text), ioutil.WriteFile(location, text, 0644)
}

func generateConfig(captiveCoreToml *CaptiveCoreToml, mode stellarCoreRunnerMode) ([]byte, error) {
	if mode == stellarCoreRunnerModeOffline {
		var err error
		captiveCoreToml, err = captiveCoreToml.CatchupToml()
		if err != nil {
			return nil, errors.Wrap(err, "could not generate catch up config")
		}
	}

	if !captiveCoreToml.QuorumSetIsConfigured() {
		return nil, errors.New("captive-core config file does not define any quorum set")
	}

	text, err := captiveCoreToml.Marshal()
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal captive core config")
	}
	return text, nil
}

func (r *stellarCoreRunner) getConfFileName() string {
	joinedPath := filepath.Join(r.storagePath, "stellar-core.conf")

	// Given that `storagePath` can be anything, we need the full, absolute path
	// here so that everything Core needs is created under the storagePath
	// subdirectory.
	//
	// If the path *can't* be absolutely resolved (bizarre), we can still try
	// recovering by using the path the user specified directly.
	path, err := filepath.Abs(joinedPath)
	if err != nil {
		r.log.Warnf("Failed to resolve %s as an absolute path: %s", joinedPath, err)
		return joinedPath
	}
	return path
}

func (r *stellarCoreRunner) getLogLineWriter() io.Writer {
	rd, wr := io.Pipe()
	br := bufio.NewReader(rd)

	// Strip timestamps from log lines from captive stellar-core. We emit our own.
	dateRx := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3} `)
	go func() {
		levelRx := regexp.MustCompile(`\[(\w+) ([A-Z]+)\] (.*)`)
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				break
			}
			line = dateRx.ReplaceAllString(line, "")
			line = strings.TrimSpace(line)

			if line == "" {
				continue
			}

			matches := levelRx.FindStringSubmatch(line)
			if len(matches) >= 4 {
				// Extract the substrings from the log entry and trim it
				category, level := matches[1], matches[2]
				line = matches[3]

				levelMapping := map[string]func(string, ...interface{}){
					"FATAL":   r.log.Errorf,
					"ERROR":   r.log.Errorf,
					"WARNING": r.log.Warnf,
					"INFO":    r.log.Infof,
					"DEBUG":   r.log.Debugf,
				}

				writer := r.log.Infof
				if f, ok := levelMapping[strings.ToUpper(level)]; ok {
					writer = f
				}
				writer("%s: %s", category, line)
			} else {
				r.log.Info(line)
			}
		}
	}()
	return wr
}

func (r *stellarCoreRunner) createCmd(params ...string) *exec.Cmd {
	allParams := append([]string{"--conf", r.getConfFileName()}, params...)
	cmd := exec.Command(r.executablePath, allParams...)
	cmd.Dir = r.storagePath
	cmd.Stdout = r.getLogLineWriter()
	cmd.Stderr = r.getLogLineWriter()
	return cmd
}

// context returns the context.Context instance associated with the running captive core instance
func (r *stellarCoreRunner) context() context.Context {
	return r.ctx
}

// catchup executes the catchup command on the captive core subprocess
func (r *stellarCoreRunner) catchup(from, to uint32) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	// check if we have already been closed
	if r.ctx.Err() != nil {
		return r.ctx.Err()
	}

	if r.started {
		return errors.New("runner already started")
	}

	rangeArg := fmt.Sprintf("%d/%d", to, to-from+1)
	params := []string{"catchup", rangeArg, "--metadata-output-stream", r.getPipeName()}

	// horizon operator has specified to use external storage for captive core ledger state
	// instruct captive core invocation to not use memory, and in that case
	// cc will look at DATABASE property in cfg toml for the external storage source to use.
	// when using external storage of ledgers, use new-db to first set the state of
	// remote db storage to genesis to purge any prior state and reset.
	if r.useDB {
		if err := r.createCmd("new-db").Run(); err != nil {
			return errors.Wrap(err, "error initializing core db")
		}
	} else {
		params = append(params, "--in-memory")
	}

	r.cmd = r.createCmd(params...)

	var err error
	r.pipe, err = r.start(r.cmd)
	if err != nil {
		r.closeLogLineWriters(r.cmd)
		return errors.Wrap(err, "error starting `stellar-core catchup` subprocess")
	}

	r.started = true
	r.ledgerBuffer = newBufferedLedgerMetaReader(r.pipe.Reader)
	go r.ledgerBuffer.start()

	if binaryWatcher, err := newFileWatcher(r); err != nil {
		r.log.Warnf("could not create captive core binary watcher: %v", err)
	} else {
		go binaryWatcher.loop()
	}

	r.wg.Add(1)
	go r.handleExit()

	return nil
}

// runFrom executes the run command with a starting ledger on the captive core subprocess
func (r *stellarCoreRunner) runFrom(from uint32, hash string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	// check if we have already been closed
	if r.ctx.Err() != nil {
		return r.ctx.Err()
	}

	if r.started {
		return errors.New("runner already started")
	}

	if r.useDB {
		if err := r.createCmd("new-db").Run(); err != nil {
			return errors.Wrap(err, "error initializing core db")
		}
		// Do a quick catch-up to set the LCL in core to be our expected starting
		// point.
		if from > 2 {
			if err := r.createCmd("catchup", fmt.Sprintf("%d/0", from-1)).Run(); err != nil {
				return errors.Wrap(err, "error runing stellar-core catchup")
			}
		} else if err := r.createCmd("catchup", "2/0").Run(); err != nil {
			return errors.Wrap(err, "error runing stellar-core catchup")
		}

		r.cmd = r.createCmd(
			"run",
			"--metadata-output-stream",
			r.getPipeName(),
		)
	} else {
		r.cmd = r.createCmd(
			"run",
			"--in-memory",
			"--start-at-ledger", fmt.Sprintf("%d", from),
			"--start-at-hash", hash,
			"--metadata-output-stream", r.getPipeName(),
		)
	}

	var err error
	r.pipe, err = r.start(r.cmd)
	if err != nil {
		r.closeLogLineWriters(r.cmd)
		return errors.Wrap(err, "error starting `stellar-core run` subprocess")
	}

	r.started = true
	r.ledgerBuffer = newBufferedLedgerMetaReader(r.pipe.Reader)
	go r.ledgerBuffer.start()

	if binaryWatcher, err := newFileWatcher(r); err != nil {
		r.log.Warnf("could not create captive core binary watcher: %v", err)
	} else {
		go binaryWatcher.loop()
	}

	r.wg.Add(1)
	go r.handleExit()

	return nil
}

func (r *stellarCoreRunner) handleExit() {
	defer r.wg.Done()

	// Pattern recommended in:
	// https://github.com/golang/go/blob/cacac8bdc5c93e7bc71df71981fdf32dded017bf/src/cmd/go/script_test.go#L1091-L1098
	interrupt := os.Interrupt
	if runtime.GOOS == "windows" {
		// Per https://golang.org/pkg/os/#Signal, “Interrupt is not implemented on
		// Windows; using it with os.Process.Signal will return an error.”
		// Fall back to Kill instead.
		interrupt = os.Kill
	}

	errc := make(chan error)
	go func() {
		select {
		case errc <- nil:
			return
		case <-r.ctx.Done():
		}

		err := r.cmd.Process.Signal(interrupt)
		if err == nil {
			err = r.ctx.Err() // Report ctx.Err() as the reason we interrupted.
		} else if err.Error() == "os: process already finished" {
			errc <- nil
			return
		}

		timer := time.NewTimer(10 * time.Second)
		select {
		// Report ctx.Err() as the reason we interrupted the process...
		case errc <- r.ctx.Err():
			timer.Stop()
			return
		// ...but after killDelay has elapsed, fall back to a stronger signal.
		case <-timer.C:
		}

		// Wait still hasn't returned.
		// Kill the process harder to make sure that it exits.
		//
		// Ignore any error: if cmd.Process has already terminated, we still
		// want to send ctx.Err() (or the error from the Interrupt call)
		// to properly attribute the signal that may have terminated it.
		_ = r.cmd.Process.Kill()

		errc <- err
	}()

	waitErr := r.cmd.Wait()
	r.closeLogLineWriters(r.cmd)

	r.lock.Lock()
	defer r.lock.Unlock()

	// By closing the pipe file we will send an EOF to the pipe reader used by ledgerBuffer.
	// We need to do this operation with the lock to ensure that the processExitError is available
	// when the ledgerBuffer channel is closed
	if closeErr := r.pipe.File.Close(); closeErr != nil {
		r.log.WithError(closeErr).Warn("could not close captive core write pipe")
	}

	r.processExited = true
	if interruptErr := <-errc; interruptErr != nil {
		r.processExitError = interruptErr
	} else {
		r.processExitError = waitErr
	}
}

// closeLogLineWriters closes the go routines created by getLogLineWriter()
func (r *stellarCoreRunner) closeLogLineWriters(cmd *exec.Cmd) {
	cmd.Stdout.(*io.PipeWriter).Close()
	cmd.Stderr.(*io.PipeWriter).Close()
}

// getMetaPipe returns a channel which contains ledgers streamed from the captive core subprocess
func (r *stellarCoreRunner) getMetaPipe() <-chan metaResult {
	return r.ledgerBuffer.getChannel()
}

// getProcessExitError returns an exit error (can be nil) of the process and a bool indicating
// if the process has exited yet
// getProcessExitError is thread safe
func (r *stellarCoreRunner) getProcessExitError() (bool, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.processExited, r.processExitError
}

// close kills the captive core process if it is still running and performs
// the necessary cleanup on the resources associated with the captive core process
// close is both thread safe and idempotent
func (r *stellarCoreRunner) close() error {
	r.lock.Lock()
	started := r.started
	storagePath := r.storagePath

	r.storagePath = ""

	// check if we have already closed
	if storagePath == "" {
		r.lock.Unlock()
		return nil
	}

	if !started {
		// Update processExited if handleExit that updates it not even started
		// (error before command run).
		r.processExited = true
	}

	r.cancel()
	r.lock.Unlock()

	// only reap captive core sub process and related go routines if we've started
	// otherwise, just cleanup the temp dir
	if started {
		// wait for the stellar core process to terminate
		r.wg.Wait()

		// drain meta pipe channel to make sure the ledger buffer goroutine exits
		for range r.getMetaPipe() {

		}

		// now it's safe to close the pipe reader
		// because the ledger buffer is no longer reading from it
		r.pipe.Reader.Close()
	}

	if runtime.GOOS == "windows" ||
		(r.processExitError != nil && r.processExitError != context.Canceled) ||
		r.mode == stellarCoreRunnerModeOffline {
		// It's impossible to send SIGINT on Windows so buckets can become
		// corrupted. If we can't reuse it, then remove it.
		// We also remove the storage path if there was an error terminating the
		// process (files can be corrupted).
		// We remove all files when reingesting to save disk space.
		return os.RemoveAll(storagePath)
	}

	return nil
}
