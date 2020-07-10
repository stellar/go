package cmd

import (
	"go/types"
	"strings"

	"github.com/google/tink/go/insecurecleartextkeyset"
	"github.com/google/tink/go/integration/awskms"
	"github.com/google/tink/go/keyset"
	tinkpb "github.com/google/tink/go/proto/tink_go_proto"
	"github.com/spf13/cobra"
	"github.com/stellar/go/support/config"
	"github.com/stellar/go/support/errors"
	supportlog "github.com/stellar/go/support/log"
)

type KeysetCreateCommand struct {
	Logger              *supportlog.Entry
	KeyTemplate         *tinkpb.KeyTemplate
	EncryptionKMSKeyURI string
}

func (c *KeysetCreateCommand) Command() *cobra.Command {
	configOpts := config.ConfigOptions{
		{
			Name:        "encryption-kms-key-uri",
			Usage:       "URI for a remote KMS key used to encrypt the Tink keyset (format: aws-kms://arn:aws:kms:<region>:<account-id>:key/<key-id>)",
			OptType:     types.String,
			ConfigKey:   &c.EncryptionKMSKeyURI,
			FlagDefault: "",
			Required:    false,
		},
	}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new Tink keyset containing a single key",
		Run: func(cmd *cobra.Command, args []string) {
			configOpts.Require()
			configOpts.SetValues()
			c.Create()
		},
	}
	configOpts.Init(cmd)

	return cmd
}

func (c *KeysetCreateCommand) Create() {
	keysetPublic, keysetPrivateCleartext, keysetPrivateEncrypted, err := createKeyset(c.EncryptionKMSKeyURI, c.KeyTemplate)
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

func createKeyset(kmsKeyURI string, keyTemplate *tinkpb.KeyTemplate) (publicCleartext string, privateCleartext string, privateEncrypted string, err error) {
	khPriv, err := keyset.NewHandle(keyTemplate)
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
			return "", "", "", errors.Wrap(err, "writing encrypted keyset private")
		}
	}

	err = insecurecleartextkeyset.Write(khPriv, keyset.NewJSONWriter(&keysetPrivateCleartext))
	if err != nil {
		return "", "", "", errors.Wrap(err, "writing cleartext keyset private")
	}

	khPub, err := khPriv.Public()
	if err != nil {
		return "", "", "", errors.Wrap(err, "getting key handle for keyset public")
	}

	err = khPub.WriteWithNoSecrets(keyset.NewJSONWriter(&keysetPublic))
	if err != nil {
		return "", "", "", errors.Wrap(err, "writing cleartext keyset public")
	}

	return keysetPublic.String(), keysetPrivateCleartext.String(), keysetPrivateEncrypted.String(), nil
}
