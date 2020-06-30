// +build !windows

package ledgerbackend

import (
	"bufio"
	"os"
	"syscall"

	"github.com/pkg/errors"
)

// Posix-specific methods for the stellarCoreRunner type.

func (c *stellarCoreRunner) getPipeName() string {
	// The exec.Cmd.ExtraFiles field carries *io.File values that are assigned
	// to child process fds counting from 3, and we'll be passing exactly one
	// fd: the write end of the anonymous pipe below.
	return "fd:3"
}

// Starts the subprocess and sets the c.metaPipe field
func (c *stellarCoreRunner) start() error {
	// First make an anonymous pipe.
	// Note io.File objects close-on-finalization.
	readFile, writeFile, e := os.Pipe()
	if e != nil {
		return errors.Wrap(e, "error making a pipe")
	}

	defer writeFile.Close()

	// Then output the config file pointing to it.
	e = c.writeConf()
	if e != nil {
		return errors.Wrap(e, "error writing conf")
	}

	// Add the write-end to the set of inherited file handles. This is defined
	// to be fd 3 on posix platforms.
	c.cmd.ExtraFiles = []*os.File{writeFile}
	e = c.cmd.Start()
	if e != nil {
		return errors.Wrap(e, "error starting stellar-core")
	}

	// Do not remove bufio.Reader wrapping. Turns out that each read from a pipe
	// adds an overhead time so it's better to preload data to a buffer.
	c.metaPipe = bufio.NewReaderSize(readFile, 1024*1024)
	return nil
}

func (c *stellarCoreRunner) processIsAlive() bool {
	return c.cmd != nil &&
		c.cmd.Process != nil &&
		c.cmd.Process.Signal(syscall.Signal(0)) == nil
}
