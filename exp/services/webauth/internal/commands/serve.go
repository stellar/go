package commands

import (
	"go/types"

	"github.com/spf13/cobra"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/exp/services/webauth/internal/serve"
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
			Name:        "horizon-url",
			Usage:       "Horizon URL used for looking up account details",
			OptType:     types.String,
			ConfigKey:   &opts.HorizonURL,
			FlagDefault: horizonclient.DefaultTestNetClient.HorizonURL,
			Required:    true,
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
			Usage:     "Stellar signing key used for signing transactions",
			OptType:   types.String,
			ConfigKey: &opts.SigningKey,
			Required:  true,
		},
		{
			Name:           "challenge-expires-in",
			Usage:          "The time period in seconds after which the challenge transaction expires",
			OptType:        types.Int,
			CustomSetValue: config.SetDuration,
			ConfigKey:      &opts.ChallengeExpiresIn,
			FlagDefault:    300,
			Required:       true,
		},
		{
			Name:      "jwt-key",
			Usage:     "Base64 encoded ECDSA private key used for signing JWTs",
			OptType:   types.String,
			ConfigKey: &opts.JWTPrivateKey,
			Required:  true,
		},
		{
			Name:           "jwt-expires-in",
			Usage:          "The time period in seconds after which the JWT expires",
			OptType:        types.Int,
			CustomSetValue: config.SetDuration,
			ConfigKey:      &opts.JWTExpiresIn,
			FlagDefault:    300,
			Required:       true,
		},
	}
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the SEP-10 Web Authentication server",
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
