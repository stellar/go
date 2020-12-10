// +build windows

package ledgerbackend

import (
	"fmt"
	"github.com/Microsoft/go-winio"
)

// Windows-specific methods for the stellarCoreRunner type.

func (c *stellarCoreRunner) getPipeName() string {
	return fmt.Sprintf(`\\.\pipe\%s`, c.nonce)
}

func (c *stellarCoreRunner) start() (pipe, error) {
	// First set up the server pipe.
	listener, err := winio.ListenPipe(c.getPipeName(), nil)
	if err != nil {
		return pipe{}, err
	}

	// Then start the process.
	err = c.cmd.Start()
	if err != nil {
		listener.Close()
		return pipe{}, err
	}

	// Then accept on the server end.
	connection, err := listener.Accept()
	if err != nil {
		listener.Close()
		return pipe{}, err
	}

	return pipe{Reader: connection, Writer: listener}, nil
}
