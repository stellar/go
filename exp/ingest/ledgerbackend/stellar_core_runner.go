package ledgerbackend

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type stellarCoreRunnerInterface interface {
	run(from, to uint32) error
	getMetaPipe() io.Reader
	close() error
}

type stellarCoreRunner struct {
	executablePath    string
	networkPassphrase string
	historyURLs       []string

	cmd      *exec.Cmd
	metaPipe io.Reader
	tempDir  string
}

func (r *stellarCoreRunner) getConf() string {
	lines := []string{
		"# Generated file -- do not edit",
		"RUN_STANDALONE=true",
		"NODE_IS_VALIDATOR=false",
		"DISABLE_XDR_FSYNC=true",
		"UNSAFE_QUORUM=true",
		fmt.Sprintf(`NETWORK_PASSPHRASE="%s"`, r.networkPassphrase),
		fmt.Sprintf(`BUCKET_DIR_PATH="%s"`, filepath.Join(r.getTmpDir(), "buckets")),
		fmt.Sprintf(`METADATA_OUTPUT_STREAM="%s"`, r.getPipeName()),
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
	return filepath.Join(r.getTmpDir(), "stellar-core.conf")
}

func (*stellarCoreRunner) getLogLineWriter() io.Writer {
	_, w := io.Pipe()
	// br := bufio.NewReader(r)
	// // Strip timestamps from log lines from captive stellar-core. We emit our own.
	// dateRx := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3} `)
	// go func() {
	// 	for {
	// 		line, e := br.ReadString('\n')
	// 		if e != nil {
	// 			break
	// 		}
	// 		line = dateRx.ReplaceAllString(line, "")
	// 		fmt.Print(line)
	// 	}
	// }()
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
	e := os.MkdirAll(dir, 0755)
	if e != nil {
		return errors.Wrap(e, "error creating subprocess tmpdir")
	}
	conf := r.getConf()
	return ioutil.WriteFile(r.getConfFileName(), []byte(conf), 0644)
}

func (r *stellarCoreRunner) run(from, to uint32) error {
	err := r.writeConf()
	if err != nil {
		return errors.Wrap(err, "error writing configuration")
	}

	rangeArg := fmt.Sprintf("%d/%d", to, to-from+1)
	args := []string{"--conf", r.getConfFileName(), "catchup", rangeArg, "--replay-in-memory"}
	cmd := exec.Command(r.executablePath, args...)
	cmd.Dir = r.getTmpDir()
	cmd.Stdout = r.getLogLineWriter()
	cmd.Stderr = cmd.Stdout
	r.cmd = cmd
	err = r.start()
	if err != nil {
		return errors.Wrap(err, "error starting stellar-core subprocess")
	}
	return nil
}

func (r *stellarCoreRunner) getMetaPipe() io.Reader {
	return r.metaPipe
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
