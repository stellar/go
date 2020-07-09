// +build windows

package ledgerbackend

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/Microsoft/go-winio"
)

// Windows-specific methods for the stellarCoreRunner type.

func (c *stellarCoreRunner) getPipeName() string {
	return fmt.Sprintf(`\\.\pipe\%s`, c.nonce)
}

func (c *stellarCoreRunner) start() (io.Reader, error) {
	// First set up the server pipe.
	listener, err := winio.ListenPipe(c.getPipeName(), nil)
	if err != nil {
		return io.Reader(nil), err
	}

	// Then write config file pointing to it.
	err = c.writeConf()
	if err != nil {
		return io.Reader(nil), err
	}

	// Then start the process.
	err = c.cmd.Start()
	if err != nil {
		return io.Reader(nil), err
	}

	// Then accept on the server end.
	connection, err := listener.Accept()
	if err != nil {
		return connection, err
	}

	// Do not remove bufio.Reader wrapping. Turns out that each read from a pipe
	// adds an overhead time so it's better to preload data to a buffer.
	return bufio.NewReaderSize(connection, 1024*1024), nil
}

func (c *stellarCoreRunner) processIsAlive() bool {
	if c.cmd == nil {
		return false
	}
	if c.cmd.Process == nil {
		return false
	}
	p, err := os.FindProcess(c.cmd.Process.Pid)
	if err != nil || p == nil {
		return false
	}
	return true
}
