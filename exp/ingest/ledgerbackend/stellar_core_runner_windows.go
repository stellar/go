// +build windows

package ledgerbackend

import (
	"bufio"
	"fmt"
	"os"

	"github.com/Microsoft/go-winio"
)

// Windows-specific methods for the stellarCoreRunner type.

func (c *stellarCoreRunner) getPipeName() string {
	return fmt.Sprintf(`\\.\pipe\%s`, c.nonce)
}

func (c *stellarCoreRunner) start() error {
	// First set up the server pipe.
	listener, e := winio.ListenPipe(c.getPipeName(), nil)
	if e != nil {
		return e
	}

	// Then write config file pointing to it.
	e = c.writeConf()
	if e != nil {
		return e
	}

	// Then start the process.
	e = c.cmd.Start()
	if e != nil {
		return e
	}

	// Then accept on the server end.
	connection, e := listener.Accept()
	if e != nil {
		return e
	}

	c.metaPipe = bufio.NewReaderSize(connection, 1024*1024)
	return nil
}

func (c *stellarCoreRunner) processIsAlive() bool {
	if c.cmd == nil {
		return false
	}
	if c.cmd.Process == nil {
		return false
	}
	p, e := os.FindProcess(c.cmd.Process.Pid)
	if e != nil || p == nil {
		return false
	}
	return true
}
