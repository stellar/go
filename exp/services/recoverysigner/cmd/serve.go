package cmd

import (
	"go/types"

	"github.com/spf13/cobra"
	"github.com/stellar/go/exp/services/recoverysigner/internal/serve"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/config"
	supportlog "github.com/stellar/go/support/log"
)

type ServeCommand struct {
	Logger *supportlog.Entry
}

func (c *ServeCommand) Command() *cobra.Command {
	opts := serve.Options{
		Logger: c.Logger,
	}
	configOpts := config.ConfigOptions{
		{
			Name:        "port",
			Usage:       "Port to listen and serve on",
			OptType:     types.Int,
			ConfigKey:   &opts.Port,
			FlagDefault: 8000,
			Required:    true,
		},
		{
			Name:        "db-url",
			Usage:       "Database URL",
			OptType:     types.String,
			ConfigKey:   &opts.DatabaseURL,
			FlagDefault: "postgres://localhost:5432/?sslmode=disable",
			Required:    false,
		},
		{
			Name:        "network-passphrase",
			Usage:       "Network passphrase of the Stellar network transactions should be signed for",
			OptType:     types.String,
			ConfigKey:   &opts.NetworkPassphrase,
			FlagDefault: network.TestNetworkPassphrase,
			Required:    true,
		},
		{
			Name:      "signing-key",
			Usage:     "Stellar signing key used for signing transactions (will be deprecated with per-account keys in the future)",
			OptType:   types.String,
			ConfigKey: &opts.SigningKey,
			Required:  true,
		},
		{
			Name:      "sep10-jwks",
			Usage:     "JSON Web Key Set (JWKS) containing exactly one key used to validate SEP-10 JWTs (if the key is an asymmetric key that has separate public and private key, the JWK need only contain the public key)",
			OptType:   types.String,
			ConfigKey: &opts.SEP10JWKS,
			Required:  true,
		},
		{
			Name:      "sep10-jwt-issuer",
			Usage:     "JWT issuer to verify is in the SEP-10 JWT iss field (not checked if empty)",
			OptType:   types.String,
			ConfigKey: &opts.SEP10JWTIssuer,
			Required:  false,
		},
		{
			Name:      "firebase-project-id",
			Usage:     "Firebase project ID to use for validating Firebase JWTs",
			OptType:   types.String,
			ConfigKey: &opts.FirebaseProjectID,
			Required:  true,
		},
	}
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the SEP-30 Recover Signer server",
		Run: func(_ *cobra.Command, _ []string) {
			configOpts.Require()
			configOpts.SetValues()
			c.Run(opts)
		},
	}
	configOpts.Init(cmd)
	return cmd
}

func (c *ServeCommand) Run(opts serve.Options) {
	serve.Serve(opts)
}
