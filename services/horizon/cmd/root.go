package cmd

import (
	"fmt"
	stdLog "log"
	"os"

	"github.com/spf13/cobra"
	horizon "github.com/stellar/go/services/horizon/internal"
)

var (
	config, flags = horizon.Flags()

	rootCmd = &cobra.Command{
		Use:   "horizon",
		Short: "client-facing api server for the stellar network",
		Long:  "client-facing api server for the stellar network. It acts as the interface between Stellar Core and applications that want to access the Stellar network. It allows you to submit transactions to the network, check the status of accounts, subscribe to event streams and more.",
		Run: func(cmd *cobra.Command, args []string) {
			horizon.NewAppFromFlags(config, flags).Serve()
		},
	}
)

func init() {
	err := flags.Init(rootCmd)
	if err != nil {
		stdLog.Fatal(err.Error())
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
