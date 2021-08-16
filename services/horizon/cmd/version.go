package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	apkg "github.com/stellar/go/support/app"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print horizon and Golang runtime version",
	Long:  "",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(apkg.Version())
		fmt.Println(runtime.Version())
		return nil
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
