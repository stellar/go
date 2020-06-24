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
	"github.com/stellar/go/support/errors"
	supportlog "github.com/stellar/go/support/log"
)

type KeysetCommand struct {
	Logger              *supportlog.Entry
	EncryptionKMSKeyURI string
}

func (c *KeysetCommand) Command() *cobra.Command {
	configOpts := config.ConfigOptions{
		{
			Name:        "encryption-kms-key-uri",
			Usage:       "URI for a remote KMS key used to encrypt Tink keyset",
			OptType:     types.String,
			ConfigKey:   &c.EncryptionKMSKeyURI,
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
	keysetPublic, keysetPrivateCleartext, keysetPrivateEncrypted, err := createKeyset(c.EncryptionKMSKeyURI)
	if err != nil {
		c.Logger.Errorf("Error creating keyset: %v", err)
		return
	}

	c.Logger.Print("Cleartext keyset public:", keysetPublic)
	c.Logger.Print("Cleartext keyset private:", keysetPrivateCleartext)

	if keysetPrivateEncrypted != "" {
		c.Logger.Print("Encrypted keyset private:", keysetPrivateEncrypted)
	}
}

func createKeyset(kmsKeyURI string) (publicCleartext string, privateCleartext string, privateEncrypted string, err error) {
	khPriv, err := keyset.NewHandle(hybrid.ECIESHKDFAES128GCMKeyTemplate())
	if err != nil {
		return "", "", "", errors.Wrap(err, "generating a new keyset")
	}

	keysetPrivateEncrypted := strings.Builder{}
	keysetPrivateCleartext := strings.Builder{}
	keysetPublic := strings.Builder{}

	if kmsKeyURI != "" {
		kmsClient, kmsErr := awskms.NewClient(kmsKeyURI)
		if kmsErr != nil {
			return "", "", "", errors.Wrap(kmsErr, "initializing AWS KMS client")
		}

		aead, kmsErr := kmsClient.GetAEAD(kmsKeyURI)
		if kmsErr != nil {
			return "", "", "", errors.Wrap(kmsErr, "getting AEAD primitive from KMS")
		}

		err = khPriv.Write(keyset.NewJSONWriter(&keysetPrivateEncrypted), aead)
		if err != nil {
			return "", "", "", errors.Wrap(err, "writing encrypted keyset containing private key")
		}
	}

	err = insecurecleartextkeyset.Write(khPriv, keyset.NewJSONWriter(&keysetPrivateCleartext))
	if err != nil {
		return "", "", "", errors.Wrap(err, "writing cleartext keyset containing private key")
	}

	khPub, err := khPriv.Public()
	if err != nil {
		return "", "", "", errors.Wrap(err, "getting keyhandle for public key")
	}

	err = khPub.WriteWithNoSecrets(keyset.NewJSONWriter(&keysetPublic))
	if err != nil {
		return "", "", "", errors.Wrap(err, "writing cleartext keyset containing public key")
	}

	return keysetPublic.String(), keysetPrivateCleartext.String(), keysetPrivateEncrypted.String(), nil
}
