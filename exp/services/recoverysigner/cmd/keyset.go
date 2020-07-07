package cmd

import (
	tinkpb "github.com/google/tink/go/proto/tink_go_proto"
	"github.com/spf13/cobra"
	supportlog "github.com/stellar/go/support/log"
)

type KeysetCommand struct {
	Logger      *supportlog.Entry
	KeyTemplate *tinkpb.KeyTemplate
}

func (c *KeysetCommand) Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "encryption-tink-keyset",
		Short: "Run Tink keyset operations",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	cmd.AddCommand((&KeysetCreateCommand{Logger: c.Logger, KeyTemplate: c.KeyTemplate}).Command())
	cmd.AddCommand((&KeysetRotateCommand{Logger: c.Logger, KeyTemplate: c.KeyTemplate}).Command())
	cmd.AddCommand((&KeysetDecryptCommand{Logger: c.Logger}).Command())
	cmd.AddCommand((&KeysetEncryptCommand{Logger: c.Logger}).Command())

	return cmd
}
