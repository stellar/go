package main

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/stellar/go/tools/stellar-hd-wallet/commands"
)

var mainCmd = &cobra.Command{
	Use:   "stellar-hd-wallet",
	Short: "Simple HD wallet for Stellar Lumens",
}

func init() {
	mainCmd.AddCommand(commands.NewCmd)
	mainCmd.AddCommand(commands.AccountsCmd)
}

func main() {
	if err := mainCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
