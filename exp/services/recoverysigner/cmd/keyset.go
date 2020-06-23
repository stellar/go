package cmd

import (
	"go/types"
	"strings"

	"github.com/google/tink/go/hybrid"
	"github.com/google/tink/go/insecurecleartextkeyset"
	"github.com/google/tink/go/integration/awskms"
	"github.com/google/tink/go/keyset"
	"github.com/spf13/cobra"
	"github.com/stellar/go/support/config"
	supportlog "github.com/stellar/go/support/log"
)

type KeysetCommand struct {
	Logger       *supportlog.Entry
	RemoteKEKURI string
}

func (c *KeysetCommand) Command() *cobra.Command {
	configOpts := config.ConfigOptions{
		{
			Name:        "encryption-kms-key-uri",
			Usage:       "URI for a remote KMS key used to encrypt Tink keyset",
			OptType:     types.String,
			ConfigKey:   &c.RemoteKEKURI,
			FlagDefault: "",
			Required:    false,
		},
	}
	cmd := &cobra.Command{
		Use:   "encryption-tink-keyset",
		Short: "Run Tink keyset operations",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			configOpts.SetValues()
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	configOpts.Init(cmd)

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new Tink keyset",
		Run: func(_ *cobra.Command, _ []string) {
			c.Create()
		},
	}
	cmd.AddCommand(createCmd)

	return cmd
}

func (c *KeysetCommand) Create() {
	khPriv, err := keyset.NewHandle(hybrid.ECIESHKDFAES128CTRHMACSHA256KeyTemplate())
	if err != nil {
		c.Logger.Errorf("Error generating a new keyset: %s", err.Error())
		return
	}

	keysetPrivateEncrypted := strings.Builder{}
	keysetPrivateCleartext := strings.Builder{}
	keysetPublic := strings.Builder{}

	if c.RemoteKEKURI != "" {
		kmsClient, err := awskms.NewClient(c.RemoteKEKURI)
		if err != nil {
			c.Logger.Errorf("Error initializing AWS KMS client: %s", err.Error())
			return
		}

		aead, err := kmsClient.GetAEAD(c.RemoteKEKURI)
		if err != nil {
			c.Logger.Errorf("Error getting AEAD primitive from KMS: %s", err.Error())
			return
		}

		err = khPriv.Write(keyset.NewJSONWriter(&keysetPrivateEncrypted), aead)
		if err != nil {
			c.Logger.Errorf("Error writing encrypted keyset containing private key: %s", err.Error())
			return
		}
	}

	err = insecurecleartextkeyset.Write(khPriv, keyset.NewJSONWriter(&keysetPrivateCleartext))
	if err != nil {
		c.Logger.Errorf("Error writing cleartext keyset containing private key: %s", err.Error())
		return
	}

	khPub, err := khPriv.Public()
	if err != nil {
		c.Logger.Errorf("Error getting keyhandle for public key: %s", err.Error())
		return
	}

	err = khPub.WriteWithNoSecrets(keyset.NewJSONWriter(&keysetPublic))
	if err != nil {
		c.Logger.Errorf("Error writing cleartext keyset containing public key: %s", err.Error())
		return
	}

	c.Logger.Print("Cleartext keyset public:", keysetPublic.String())
	c.Logger.Print("Cleartext keyset private:", keysetPrivateCleartext.String())
	if c.RemoteKEKURI != "" {
		c.Logger.Print("Encrypted keyset private:", keysetPrivateEncrypted.String())
	}
}
