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
	"path/filepath"
	"regexp"
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
	logPath           string
	executablePath    string
	configAppendPath  string
	networkPassphrase string
	historyURLs       []string
	httpPort          uint
	mode              stellarCoreRunnerMode

	started      bool
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	ledgerBuffer *bufferedLedgerMetaReader
	pipe         pipe

	lock             sync.Mutex
	processExited    bool
	processExitError error

	tempDir string
	nonce   string

	log *log.Entry
}

func newStellarCoreRunner(config CaptiveCoreConfig, mode stellarCoreRunnerMode) (*stellarCoreRunner, error) {
	// Create temp dir
	tempDir, err := ioutil.TempDir("", "captive-stellar-core")
	if err != nil {
		return nil, errors.Wrap(err, "error creating subprocess tmpdir")
	}

	ctx, cancel := context.WithCancel(config.Context)

	runner := &stellarCoreRunner{
		logPath:           config.LogPath,
		executablePath:    config.BinaryPath,
		configAppendPath:  config.ConfigAppendPath,
		networkPassphrase: config.NetworkPassphrase,
		historyURLs:       config.HistoryArchiveURLs,
		httpPort:          config.HTTPPort,
		mode:              mode,
		ctx:               ctx,
		cancel:            cancel,
		tempDir:           tempDir,
		nonce: fmt.Sprintf(
			"captive-stellar-core-%x",
			rand.New(rand.NewSource(time.Now().UnixNano())).Uint64(),
		),
		log: config.Log,
	}

	if err := runner.writeConf(); err != nil {
		return nil, errors.Wrap(err, "error writing configuration")
	}

	return runner, nil
}

func (r *stellarCoreRunner) generateConfig() (string, error) {
	if r.mode == stellarCoreRunnerModeOnline && r.configAppendPath == "" {
		return "", errors.New("stellar-core append config file path cannot be empty in online mode")
	}
	lines := []string{
		"# Generated file -- do not edit",
		"NODE_IS_VALIDATOR=false",
		"DISABLE_XDR_FSYNC=true",
		fmt.Sprintf(`NETWORK_PASSPHRASE="%s"`, r.networkPassphrase),
		fmt.Sprintf(`BUCKET_DIR_PATH="%s"`, filepath.Join(r.tempDir, "buckets")),
		fmt.Sprintf(`HTTP_PORT=%d`, r.httpPort),
		fmt.Sprintf(`LOG_FILE_PATH="%s"`, r.logPath),
	}

	if r.mode == stellarCoreRunnerModeOffline {
		// In offline mode, there is no need to connect to peers
		lines = append(lines, "RUN_STANDALONE=true")
		// We don't need consensus when catching up
		lines = append(lines, "UNSAFE_QUORUM=true")
	}

	if r.mode == stellarCoreRunnerModeOffline && r.configAppendPath == "" {
		// Add a fictional quorum -- necessary to convince core to start up;
		// but not used at all for our purposes. Pubkey here is just random.
		lines = append(lines,
			"[QUORUM_SET]",
			"THRESHOLD_PERCENT=100",
			`VALIDATORS=["GCZBOIAY4HLKAJVNJORXZOZRAY2BJDBZHKPBHZCRAIUR5IHC2UHBGCQR"]`)
	}

	result := strings.ReplaceAll(strings.Join(lines, "\n"), `\`, `\\`) + "\n\n"
	if r.configAppendPath != "" {
		appendConfigContents, err := ioutil.ReadFile(r.configAppendPath)
		if err != nil {
			return "", errors.Wrap(err, "reading quorum config file")
		}
		result += string(appendConfigContents) + "\n\n"
	}

	lines = []string{}
	for i, val := range r.historyURLs {
		lines = append(lines, fmt.Sprintf("[HISTORY.h%d]", i))
		lines = append(lines, fmt.Sprintf(`get="curl -sf %s/{0} -o {1}"`, val))
	}
	result += strings.Join(lines, "\n")

	return result, nil
}

func (r *stellarCoreRunner) getConfFileName() string {
	return filepath.Join(r.tempDir, "stellar-core.conf")
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
				}

				if writer, ok := levelMapping[strings.ToUpper(level)]; ok {
					writer("%s: %s", category, line)
				} else {
					r.log.Info(line)
				}
			} else {
				r.log.Info(line)
			}
		}
	}()
	return wr
}

// Makes the temp directory and writes the config file to it; called by the
// platform-specific captiveStellarCore.Start() methods.
func (r *stellarCoreRunner) writeConf() error {
	conf, err := r.generateConfig()
	if err != nil {
		return err
	}
	r.log.Debugf("captive core config file contents:\n%s", conf)
	return ioutil.WriteFile(r.getConfFileName(), []byte(conf), 0644)
}

func (r *stellarCoreRunner) createCmd(params ...string) *exec.Cmd {
	allParams := append([]string{"--conf", r.getConfFileName()}, params...)
	cmd := exec.CommandContext(r.ctx, r.executablePath, allParams...)
	cmd.Dir = r.tempDir
	cmd.Stdout = r.getLogLineWriter()
	cmd.Stderr = r.getLogLineWriter()
	return cmd
}

func (r *stellarCoreRunner) runCmd(params ...string) error {
	cmd := r.createCmd(params...)
	defer r.closeLogLineWriters(cmd)

	if err := cmd.Start(); err != nil {
		return errors.Wrapf(err, "could not start `stellar-core %v` cmd", params)
	}

	if err := cmd.Wait(); err != nil {
		return errors.Wrapf(err, "error waiting for `stellar-core %v` subprocess", params)
	}
	return nil
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
	if err := r.runCmd("new-db"); err != nil {
		return errors.Wrap(err, "error waiting for `stellar-core new-db` subprocess")
	}

	rangeArg := fmt.Sprintf("%d/%d", to, to-from+1)
	cmd := r.createCmd(
		"catchup", rangeArg,
		"--metadata-output-stream", r.getPipeName(),
		"--replay-in-memory",
	)

	var err error
	r.pipe, err = r.start(cmd)
	if err != nil {
		r.closeLogLineWriters(cmd)
		return errors.Wrap(err, "error starting `stellar-core catchup` subprocess")
	}

	r.started = true
	r.ledgerBuffer = newBufferedLedgerMetaReader(r.pipe.Reader)
	go r.ledgerBuffer.start()
	r.wg.Add(1)
	go r.handleExit(cmd)

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

	cmd := r.createCmd(
		"run",
		"--in-memory",
		"--start-at-ledger", fmt.Sprintf("%d", from),
		"--start-at-hash", hash,
		"--metadata-output-stream", r.getPipeName(),
	)

	var err error
	r.pipe, err = r.start(cmd)
	if err != nil {
		r.closeLogLineWriters(cmd)
		return errors.Wrap(err, "error starting `stellar-core run` subprocess")
	}

	r.started = true
	r.ledgerBuffer = newBufferedLedgerMetaReader(r.pipe.Reader)
	go r.ledgerBuffer.start()
	r.wg.Add(1)
	go r.handleExit(cmd)

	return nil
}

func (r *stellarCoreRunner) handleExit(cmd *exec.Cmd) {
	defer r.wg.Done()
	exitErr := cmd.Wait()
	r.closeLogLineWriters(cmd)

	r.lock.Lock()
	defer r.lock.Unlock()

	// By closing the pipe file we will send an EOF to the pipe reader used by ledgerBuffer.
	// We need to do this operation with the lock to ensure that the processExitError is available
	// when the ledgerBuffer channel is closed
	if closeErr := r.pipe.File.Close(); closeErr != nil {
		r.log.WithError(closeErr).Warn("could not close captive core write pipe")
	}

	r.processExited = true
	r.processExitError = exitErr
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
	tempDir := r.tempDir

	r.tempDir = ""

	// check if we have already closed
	if tempDir == "" {
		r.lock.Unlock()
		return nil
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

	return os.RemoveAll(tempDir)
}
