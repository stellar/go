package main

import (
	"github.com/google/tink/go/hybrid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/stellar/go/exp/services/recoverysigner/cmd"
	supportlog "github.com/stellar/go/support/log"
)

func main() {
	logger := supportlog.New()
	logger.Logger.Level = logrus.TraceLevel

	rootCmd := &cobra.Command{
		Use:   "recoverysigner [command]",
		Short: "SEP-30 Recovery Signer server",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	rootCmd.AddCommand((&cmd.ServeCommand{Logger: logger}).Command())
	rootCmd.AddCommand((&cmd.DBCommand{Logger: logger}).Command())

	// Key template used for the generation of new keys and keysets.
	keyTemplate := hybrid.ECIESHKDFAES128GCMKeyTemplate()
	rootCmd.AddCommand((&cmd.KeysetCommand{Logger: logger, KeyTemplate: keyTemplate}).Command())

	err := rootCmd.Execute()
	if err != nil {
		logger.Fatal(err)
	}
}
