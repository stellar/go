package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/stellar/go/exp/support/jwtkey"
	supportlog "github.com/stellar/go/support/log"
	"gopkg.in/square/go-jose.v2"
)

type ConvJWTKeyToJWKCommand struct {
	Logger *supportlog.Entry
}

func (c *ConvJWTKeyToJWKCommand) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convjwtkey2jwk [jwt-key]",
		Short: "Convert a JWT ECDSA private key ASN.1 DER Base64 encoded that was generated with the old genjwtkey command to a JSON Web Key",
		Run: func(_ *cobra.Command, args []string) {
			c.Run(args)
		},
	}
	return cmd
}

func (c *ConvJWTKeyToJWKCommand) Run(args []string) {
	if len(args) != 1 {
		c.Logger.Fatal("One key (ASN.1 DER Base64 encoded) must be provided.")
	}

	k, err := jwtkey.PrivateKeyFromString(args[0])
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
