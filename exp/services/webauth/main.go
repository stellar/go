package main

import (
	"log"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/stellar/go/exp/services/webauth/internal/commands"
	supportlog "github.com/stellar/go/support/log"
)

func main() {
	logger := supportlog.New()
	logger.Logger.Level = logrus.TraceLevel

	rootCmd := &cobra.Command{
		Use:   "webauth [command]",
		Short: "SEP-10 Web Authentication Server",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	rootCmd.AddCommand((&commands.ServeCommand{Logger: logger}).Command())
	rootCmd.AddCommand((&commands.GenJWTKeyCommand{Logger: logger}).Command())

	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
