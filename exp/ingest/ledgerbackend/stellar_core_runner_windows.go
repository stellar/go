// +build windows

package ledgerbackend

import (
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
	listener, e := winio.ListenPipe(c.getPipeName(), nil)
	if e != nil {
		return io.Reader(nil), e
	}

	// Then write config file pointing to it.
	e = c.writeConf()
	if e != nil {
		return io.Reader(nil), e
	}

	// Then start the process.
	e = c.cmd.Start()
	if e != nil {
		return io.Reader(nil), e
	}

	// Launch a goroutine to reap immediately on exit (I think this is right,
	// as we do not want zombies and we might abruptly forget / kill / close
	// the process, but I'm not certain).
	cmd := c.cmd
	go func() {
		cmd.Wait()
	}()

	// Then accept on the server end.
	connection, e := listener.Accept()
	if e != nil {
		return connection, e
	}

	return connection, nil
}

func (c *captiveStellarCore) processIsAlive() bool {
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
