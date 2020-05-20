// +build windows

package ledgerbackend

import (
	"bufio"
	"fmt"
	"github.com/Microsoft/go-winio"
	"os"
)

// Windows-specific methods for the captiveStellarCore type.

func (c *captiveStellarCore) getPipeName() string {
	return fmt.Sprintf(`\\.\pipe\%s`, c.nonce)
}

func (c *captiveStellarCore) start() error {
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
		return e
	}

	c.metaPipe = bufio.NewReaderSize(connection, 1024*1024)
	return nil
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
