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
			Name:      "sep10-jwt-public-key",
			Usage:     "Base64 encoded ECDSA public key used to validate SEP-10 JWTs",
			OptType:   types.String,
			ConfigKey: &opts.SEP10JWTPublicKey,
			Required:  true,
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
		Short: "Run the SEP-XX Recover Signer server",
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
