package cmd

import (
	"go/types"

	"github.com/spf13/cobra"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/network"
	"github.com/stellar/go/services/regulated-assets-approval-server/internal/assetissuer"
	"github.com/stellar/go/support/config"
)

type ConfigureAssetIssuer struct{}

func (c *ConfigureAssetIssuer) Command() *cobra.Command {
	opts := assetissuer.Options{}
	configOpts := config.ConfigOptions{
		{
			Name:      "account-issuer-secret",
			Usage:     "Secret key of the asset issuer's stellar account.",
			OptType:   types.String,
			ConfigKey: &opts.AccountIssuerSecret,
			Required:  true,
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
	}
	cmd := &cobra.Command{
		Use:   "configure-asset-issuer",
		Short: "Configure asset issuer to use SEP-8 regulated assets.",
		Run: func(_ *cobra.Command, _ []string) {
			configOpts.Require()
			configOpts.SetValues()
			c.Run(opts)
		},
	}
	configOpts.Init(cmd)
	return cmd
}

func (c *ConfigureAssetIssuer) Run(opts assetissuer.Options) {
	assetissuer.Configure(opts)
}
