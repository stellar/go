// +build !windows

package ledgerbackend

import (
	"bufio"
	"os"
	"syscall"

	"github.com/pkg/errors"
)

// Posix-specific methods for the captiveStellarCore type.

func (c *captiveStellarCore) getPipeName() string {
	// The exec.Cmd.ExtraFiles field carries *io.File values that are assigned
	// to child process fds counting from 3, and we'll be passing exactly one
	// fd: the write end of the anonymous pipe below.
	return "fd:3"
}

// Starts the subprocess and sets the c.metaPipe field
func (c *captiveStellarCore) start() error {

	// First make an anonymous pipe.
	// Note io.File objects close-on-finalization.
	readFile, writeFile, e := os.Pipe()
	if e != nil {
		return errors.Wrap(e, "error making a pipe")
	}

	defer writeFile.Close()

	// Then write config file pointing to it.
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

	// Launch a goroutine to reap immediately on exit (I think this is right,
	// as we do not want zombies and we might abruptly forget / kill / close
	// the process, but I'm not certain).
	cmd := c.cmd
	go cmd.Wait()

	c.metaPipe = bufio.NewReaderSize(readFile, 1024*1024)
	return nil
}

func (c *captiveStellarCore) processIsAlive() bool {
	if c.cmd == nil {
		return false
	}
	if c.cmd.Process == nil {
		return false
	}
	return c.cmd.Process.Signal(syscall.Signal(0)) == nil
}
