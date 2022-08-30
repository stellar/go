package main

import (
	"github.com/spf13/cobra"

	"github.com/stellar/go/support/log"
)

func main() {
	log.SetLevel(log.InfoLevel)

	cmd := &cobra.Command{
		Use:     "tools",
		Long:    "Please specify a subcommand to use this toolset.",
		Example: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			// require a subcommand - this is just a "category"
			return cmd.Help()
		},
	}

	cmd = addCacheCommands(cmd)
	cmd = addIndexCommands(cmd)
	cmd.Execute()
}
