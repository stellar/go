package cmd

import (
	"github.com/spf13/cobra"
	horizon "github.com/stellar/go/services/horizon/internal"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "run horizon server",
	Long:  "serve initializes then starts the horizon HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		horizon.NewAppFromFlags(config, flags).Serve()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
