// +build !windows

package ledgerbackend

import (
	"io"
	"os"
	"syscall"

	"github.com/pkg/errors"
)

// Posix-specific methods for the StellarCoreRunner type.

func (c *stellarCoreRunner) getPipeName() string {
	// The exec.Cmd.ExtraFiles field carries *io.File values that are assigned
	// to child process fds counting from 3, and we'll be passing exactly one
	// fd: the write end of the anonymous pipe below.
	return "fd:3"
}

func (c *stellarCoreRunner) start() (io.Reader, error) {
	// First make an anonymous pipe.
	// Note io.File objects close-on-finalization.
	readFile, writeFile, err := os.Pipe()
	if err != nil {
		return readFile, errors.Wrap(err, "error making a pipe")
	}

	defer writeFile.Close()

	// Add the write-end to the set of inherited file handles. This is defined
	// to be fd 3 on posix platforms.
	c.cmd.ExtraFiles = []*os.File{writeFile}
	err = c.cmd.Start()
	if err != nil {
		return readFile, errors.Wrap(err, "error starting stellar-core")
	}

	c.wg.Add(1)
	go func() {
		err := make(chan error, 1)
		select {
		case err <- c.cmd.Wait():
			c.processExitError = <-err
			close(c.processExit)
			close(err)
		case <-c.shutdown:
		}
		c.wg.Done()
	}()

	return readFile, nil
}

func (c *stellarCoreRunner) processIsAlive() bool {
	return c.cmd != nil &&
		c.cmd.Process != nil &&
		c.cmd.Process.Signal(syscall.Signal(0)) == nil
}
