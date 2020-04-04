package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/stellar/go/exp/support/jwtkey"
	supportlog "github.com/stellar/go/support/log"
	"gopkg.in/square/go-jose.v2"
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

	alg := jose.ES256

	{
		jwk := jose.JSONWebKey{Key: &k.PublicKey, Algorithm: string(alg)}
		bytes, err := json.Marshal(jwk)
		if err == nil {
			c.Logger.Print("Public:", string(bytes))
		} else {
			c.Logger.Print("Public:", err)
		}
	}

	{
		jwk := jose.JSONWebKey{Key: k, Algorithm: string(alg)}
		bytes, err := json.Marshal(jwk)
		if err == nil {
			c.Logger.Print("Private:", string(bytes))
		} else {
			c.Logger.Print("Private:", err)
		}
	}
}
