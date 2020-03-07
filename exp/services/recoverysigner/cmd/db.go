package cmd

import (
	"github.com/spf13/cobra"
	supportlog "github.com/stellar/go/support/log"
)

type DBCommand struct {
	Logger *supportlog.Entry
}

func (c *DBCommand) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Run database operations",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	cmd.AddCommand((&DBMigrateCommand{Logger: c.Logger}).Command())
	return cmd
}
