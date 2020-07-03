package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stellar/go/keypair"
)

func main() {
	cmd := NewRootCmd()
	err := cmd.Execute()
	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "Error:", err)
	}
}

func NewRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stellar-key-gen",
		Short: "Generate a Stellar key.",
		Run:   gen,
	}
}

func gen(cmd *cobra.Command, args []string) {
	key, err := keypair.Random()
	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "Error:", err)
		return
	}
	fmt.Fprintln(cmd.ErrOrStderr(), "Public Key:")
	fmt.Fprintln(cmd.ErrOrStderr(), key.Address())
	fmt.Fprintln(cmd.ErrOrStderr(), "Secret Key:")
	fmt.Fprint(cmd.OutOrStdout(), key.Seed())
}
