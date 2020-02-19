package main

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
<<<<<<< HEAD
	"github.com/stellar/go/exp/services/recoverysigner/internal/commands"
=======
	"github.com/stellar/go/exp/services/recoverysigner/cmd"
>>>>>>> master
	supportlog "github.com/stellar/go/support/log"
)

func main() {
	logger := supportlog.New()
	logger.Logger.Level = logrus.TraceLevel

	rootCmd := &cobra.Command{
		Use:   "recoverysigner [command]",
		Short: "SEP-XX Recovery Signer server",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

<<<<<<< HEAD
	rootCmd.AddCommand((&commands.ServeCommand{Logger: logger}).Command())
=======
	rootCmd.AddCommand((&cmd.ServeCommand{Logger: logger}).Command())
>>>>>>> master

	err := rootCmd.Execute()
	if err != nil {
		logger.Fatal(err)
	}
}
