package main

import (
	"fmt"
	"go/format"
	"os"

	"github.com/spf13/cobra"
	"github.com/stellar/go/xdr"
)

var (
	typ string
)

var rootCmd = &cobra.Command{
	Use:   "xdr2go [base64-encoded XDR object]",
	Short: "xdr2go transforms base64 encoded XDR objects into a pretty Go code",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd, args)
	},
}

func main() {
	rootCmd.Flags().StringVarP(&typ, "type", "t", "TransactionEnvelope", "xdr type, currently only TransactionEnvelope is available")
	rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		printHelpExit(cmd, "Exactly one command argument with XDR object is required.")
	}
	var object interface{}
	switch typ {
	case "TransactionEnvelope":
		object = &xdr.TransactionEnvelope{}
	default:
		printHelpExit(cmd, "Unknown type.")
	}
	err := xdr.SafeUnmarshalBase64(args[0], object)
	if err != nil {
		panic(err)
	}

	source := fmt.Sprintf("%#v\n", object)
	formatted, err := format.Source([]byte(source))
	if err != nil {
		panic(err)
	}
	fmt.Println(string(formatted))
}

func printHelpExit(cmd *cobra.Command, msg string) {
	fmt.Println(msg)
	cmd.Help()
	os.Exit(-1)
}
