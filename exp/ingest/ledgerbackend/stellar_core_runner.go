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

func newStellarCoreRunner(executablePath, networkPassphrase string, historyURLs []string) *stellarCoreRunner {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &stellarCoreRunner{
		executablePath:    executablePath,
		networkPassphrase: networkPassphrase,
		historyURLs:       historyURLs,
		processExit:       make(chan error),
		nonce:             fmt.Sprintf("captive-stellar-core-%x", r.Uint64()),
	}
}

func (r *stellarCoreRunner) getConf() string {
	lines := []string{
		"# Generated file -- do not edit",
		fmt.Sprintf(`NETWORK_PASSPHRASE="%s"`, r.networkPassphrase),
		fmt.Sprintf(`BUCKET_DIR_PATH="%s"`, filepath.Join(r.getTmpDir(), "buckets")),
		fmt.Sprintf(`METADATA_OUTPUT_STREAM="%s"`, r.getPipeName()),
		// "RUN_STANDALONE=true",
		"NODE_IS_VALIDATOR=false",
		// "DISABLE_XDR_FSYNC=true",
		`DATABASE="postgresql://dbname=core host=localhost user=Bartek"`,

		`
[[HOME_DOMAINS]]
HOME_DOMAIN="testnet.stellar.org"
QUALITY="HIGH"

[[VALIDATORS]]
NAME="sdf_testnet_1"
HOME_DOMAIN="testnet.stellar.org"
PUBLIC_KEY="GDKXE2OZMJIPOSLNA6N6F2BVCI3O777I2OOC4BV7VOYUEHYX7RTRYA7Y"
ADDRESS="core-testnet1.stellar.org"
HISTORY="curl -sf http://history.stellar.org/prd/core-testnet/core_testnet_001/{0} -o {1}"

[[VALIDATORS]]
NAME="sdf_testnet_2"
HOME_DOMAIN="testnet.stellar.org"
PUBLIC_KEY="GCUCJTIYXSOXKBSNFGNFWW5MUQ54HKRPGJUTQFJ5RQXZXNOLNXYDHRAP"
ADDRESS="core-testnet2.stellar.org"
HISTORY="curl -sf http://history.stellar.org/prd/core-testnet/core_testnet_002/{0} -o {1}"

[[VALIDATORS]]
NAME="sdf_testnet_3"
HOME_DOMAIN="testnet.stellar.org"
PUBLIC_KEY="GC2V2EFSXN6SQTWVYA5EPJPBWWIMSD2XQNKUOHGEKB535AQE2I6IXV2Z"
ADDRESS="core-testnet3.stellar.org"
HISTORY="curl -sf http://history.stellar.org/prd/core-testnet/core_testnet_003/{0} -o {1}"

`,
	}
	for i, val := range r.historyURLs {
		lines = append(lines, fmt.Sprintf("[HISTORY.h%d]", i))
		lines = append(lines, fmt.Sprintf(`get="curl -sf %s/{0} -o {1}"`, val))
	}
	return strings.ReplaceAll(strings.Join(lines, "\n"), "\\", "\\\\")
}

func (r *stellarCoreRunner) getConfFileName() string {
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
	dir := r.getTmpDir()
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return errors.Wrap(err, "error creating subprocess tmpdir")
	}
	conf := r.getConf()
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
	if err := r.writeConf(); err != nil {
		return errors.Wrap(err, "error writing configuration")
	}

	if err := r.runCmd("new-db"); err != nil {
		return errors.Wrap(err, "error waiting for `stellar-core new-db` subprocess")
	}

	rangeArg := fmt.Sprintf("%d/%d", to, to-from+1)
	cmd, err := r.createCmd("catchup", rangeArg, "--replay-in-memory")
	if err != nil {
		return errors.Wrap(err, "error creating `stellar-core catchup` subprocess")
	}
	r.cmd = cmd
	r.metaPipe, err = r.start()
	if err != nil {
		return errors.Wrap(err, "error starting `stellar-core catchup` subprocess")
	}

	return nil
}

func (r *stellarCoreRunner) runFrom(from uint32) error {
	err := r.writeConf()
	if err != nil {
		return errors.Wrap(err, "error writing configuration")
	}

	if err = r.runCmd("new-db"); err != nil {
		return errors.Wrap(err, "error waiting for `stellar-core new-db` subprocess")
	}

	// catchup to `from` ledger
	if err = r.runCmd("catchup", fmt.Sprintf("%d/0", from-1)); err != nil {
		return errors.Wrap(err, "error running `stellar-core catchup` subprocess")
	}

	r.cmd, err = r.createCmd("run")
	if err != nil {
		return errors.Wrap(err, "error creating `stellar-core run` subprocess")
	}
	r.metaPipe, err = r.start()
	if err != nil {
		return errors.Wrap(err, "error starting `stellar-core run` subprocess")
	}

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
