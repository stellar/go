package ledgerbackend

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type stellarCoreRunnerInterface interface {
	catchup(from, to uint32) error
	runFrom(from uint32) error
	getMetaPipe() io.Reader
	getProcessExitChan() <-chan error
	close() error
}

type stellarCoreRunner struct {
	executablePath    string
	configPath        string
	networkPassphrase string
	historyURLs       []string

	cmd *exec.Cmd
	// processExit channel receives an error when the process exited with an error
	// or nil if the process exited without an error.
	processExit chan error
	metaPipe    io.Reader
	tempDir     string
	nonce       string
}

func newStellarCoreRunner(executablePath, configPath, networkPassphrase string, historyURLs []string) (*stellarCoreRunner, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	runner := &stellarCoreRunner{
		executablePath:    executablePath,
		configPath:        configPath,
		networkPassphrase: networkPassphrase,
		historyURLs:       historyURLs,
		processExit:       make(chan error),
		nonce:             fmt.Sprintf("captive-stellar-core-%x", r.Uint64()),
	}

	// Create temp dir
	dir := runner.getTmpDir()
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, errors.Wrap(err, "error creating subprocess tmpdir")
	}

	if configPath == "" {
		err := runner.writeConf()
		if err != nil {
			return nil, errors.Wrap(err, "error writing configuration")
		}
	}

	return runner, nil
}

func (r *stellarCoreRunner) generateConfig() string {
	lines := []string{
		"# Generated file -- do not edit",
		"RUN_STANDALONE=true",
		"NODE_IS_VALIDATOR=false",
		"DISABLE_XDR_FSYNC=true",
		"UNSAFE_QUORUM=true",
		fmt.Sprintf(`NETWORK_PASSPHRASE="%s"`, r.networkPassphrase),
		fmt.Sprintf(`BUCKET_DIR_PATH="%s"`, filepath.Join(r.getTmpDir(), "buckets")),
	}
	for i, val := range r.historyURLs {
		lines = append(lines, fmt.Sprintf("[HISTORY.h%d]", i))
		lines = append(lines, fmt.Sprintf(`get="curl -sf %s/{0} -o {1}"`, val))
	}
	// Add a fictional quorum -- necessary to convince core to start up;
	// but not used at all for our purposes. Pubkey here is just random.
	lines = append(lines,
		"[QUORUM_SET]",
		"THRESHOLD_PERCENT=100",
		`VALIDATORS=["GCZBOIAY4HLKAJVNJORXZOZRAY2BJDBZHKPBHZCRAIUR5IHC2UHBGCQR"]`)
	return strings.ReplaceAll(strings.Join(lines, "\n"), "\\", "\\\\")
}

func (r *stellarCoreRunner) getConfFileName() string {
	if r.configPath != "" {
		return r.configPath
	}
	return filepath.Join(r.getTmpDir(), "stellar-core.conf")
}

func (*stellarCoreRunner) getLogLineWriter() io.Writer {
	r, w := io.Pipe()
	br := bufio.NewReader(r)
	// Strip timestamps from log lines from captive stellar-core. We emit our own.
	dateRx := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3} `)
	go func() {
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				break
			}
			line = dateRx.ReplaceAllString(line, "")
			fmt.Print(line)
		}
	}()
	return w
}

func (r *stellarCoreRunner) getTmpDir() string {
	if r.tempDir != "" {
		return r.tempDir
	}
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.tempDir = filepath.Join(os.TempDir(), fmt.Sprintf("captive-stellar-core-%x", random.Uint64()))
	return r.tempDir
}

// Makes the temp directory and writes the config file to it; called by the
// platform-specific captiveStellarCore.Start() methods.
func (r *stellarCoreRunner) writeConf() error {
	conf := r.generateConfig()
	return ioutil.WriteFile(r.getConfFileName(), []byte(conf), 0644)
}

func (r *stellarCoreRunner) createCmd(params ...string) (*exec.Cmd, error) {
	allParams := append([]string{"--conf", r.getConfFileName()}, params...)
	cmd := exec.Command(r.executablePath, allParams...)
	cmd.Dir = r.getTmpDir()
	cmd.Stdout = r.getLogLineWriter()
	cmd.Stderr = r.getLogLineWriter()
	return cmd, nil
}

func (r *stellarCoreRunner) runCmd(params ...string) error {
	cmd, err := r.createCmd(params...)
	if err != nil {
		return errors.Wrapf(err, "could not create `stellar-core %v` cmd", params)
	}

	if err = cmd.Start(); err != nil {
		return errors.Wrapf(err, "could not start `stellar-core %v` cmd", params)
	}

	if err = cmd.Wait(); err != nil {
		return errors.Wrapf(err, "error waiting for `stellar-core %v` subprocess", params)
	}
	return nil
}

func (r *stellarCoreRunner) catchup(from, to uint32) error {
	if err := r.runCmd("new-db"); err != nil {
		return errors.Wrap(err, "error waiting for `stellar-core new-db` subprocess")
	}

	rangeArg := fmt.Sprintf("%d/%d", to, to-from+1)
	cmd, err := r.createCmd(
		"catchup", rangeArg,
		"--metadata-output-stream", r.getPipeName(),
		"--replay-in-memory",
	)
	if err != nil {
		return errors.Wrap(err, "error creating `stellar-core catchup` subprocess")
	}
	r.cmd = cmd
	r.metaPipe, err = r.start()
	if err != nil {
		return errors.Wrap(err, "error starting `stellar-core catchup` subprocess")
	}

	// Do not remove bufio.Reader wrapping. Turns out that each read from a pipe
	// adds an overhead time so it's better to preload data to a buffer.
	r.metaPipe = bufio.NewReaderSize(r.metaPipe, 1024*1024)
	return nil
}

func (r *stellarCoreRunner) runFrom(from uint32) error {
	var err error
	if err = r.runCmd("new-db"); err != nil {
		return errors.Wrap(err, "error waiting for `stellar-core new-db` subprocess")
	}

	// catchup to `from` ledger
	if err = r.runCmd("catchup", fmt.Sprintf("%d/0", from-1)); err != nil {
		return errors.Wrap(err, "error running `stellar-core catchup` subprocess")
	}

	r.cmd, err = r.createCmd(
		"run",
		"--metadata-output-stream", r.getPipeName(),
	)
	if err != nil {
		return errors.Wrap(err, "error creating `stellar-core run` subprocess")
	}
	r.metaPipe, err = r.start()
	if err != nil {
		return errors.Wrap(err, "error starting `stellar-core run` subprocess")
	}

	// Do not remove bufio.Reader wrapping. Turns out that each read from a pipe
	// adds an overhead time so it's better to preload data to a buffer.
	r.metaPipe = bufio.NewReaderSize(r.metaPipe, 1024*1024)
	return nil
}

func (r *stellarCoreRunner) getMetaPipe() io.Reader {
	return r.metaPipe
}

func (r *stellarCoreRunner) getProcessExitChan() <-chan error {
	return r.processExit
}

func (r *stellarCoreRunner) close() error {
	var err1, err2 error

	if r.processIsAlive() {
		err1 = r.cmd.Process.Kill()
		r.cmd.Wait()
		r.cmd = nil
	}
	err2 = os.RemoveAll(r.getTmpDir())
	if err1 != nil {
		return errors.Wrap(err1, "error killing subprocess")
	}
	if err2 != nil {
		return errors.Wrap(err2, "error removing subprocess tmpdir")
	}
	return nil
}
