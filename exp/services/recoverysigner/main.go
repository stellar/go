package main

import (
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

	err := rootCmd.Execute()
	if err != nil {
		logger.Fatal(err)
	}
}
