package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	horizonclient "github.com/stellar/go/exp/clients/horizon"
	demo "github.com/stellar/go/exp/txnbuild/cmd/demo/operations"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create and fund some demo accounts on the TestNet",
	Long:  `This command creates four test accounts for use with further operations.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Initialising TestNet accounts...")
		keys := demo.InitKeys()
		client := horizonclient.DefaultTestNetClient

		demo.Initialise(client, keys)
		fmt.Println("Initialisation complete.")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
