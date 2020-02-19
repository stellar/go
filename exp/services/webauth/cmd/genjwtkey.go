package cmd

import (
	"github.com/spf13/cobra"
	"github.com/stellar/go/exp/support/jwtkey"
	supportlog "github.com/stellar/go/support/log"
)

type GenJWTKeyCommand struct {
	Logger *supportlog.Entry
}

func (c *GenJWTKeyCommand) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "genjwtkey",
		Short: "Generate a JWT ECDSA key",
		Run: func(_ *cobra.Command, _ []string) {
			c.Run()
		},
	}
	return cmd
}

func (c *GenJWTKeyCommand) Run() {
	k, err := jwtkey.GenerateKey()
	if err != nil {
		c.Logger.Fatal(err)
	}

	if public, err := jwtkey.PublicKeyToString(&k.PublicKey); err == nil {
		c.Logger.Print("Public:", public)
	} else {
		c.Logger.Print("Public:", err)
	}

	if private, err := jwtkey.PrivateKeyToString(k); err == nil {
		c.Logger.Print("Private:", private)
	} else {
		c.Logger.Print("Private:", err)
	}
}
