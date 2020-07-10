package cmd

import (
	"go/types"
	"strings"

	"github.com/google/tink/go/insecurecleartextkeyset"
	"github.com/google/tink/go/integration/awskms"
	"github.com/google/tink/go/keyset"
	"github.com/spf13/cobra"
	"github.com/stellar/go/support/config"
	"github.com/stellar/go/support/errors"
	supportlog "github.com/stellar/go/support/log"
)

type KeysetDecryptCommand struct {
	Logger                   *supportlog.Entry
	EncryptionKMSKeyURI      string
	EncryptionTinkKeysetJSON string
}

func (c *KeysetDecryptCommand) Command() *cobra.Command {
	configOpts := config.ConfigOptions{
		{
			Name:        "encryption-kms-key-uri",
			Usage:       "URI for a remote KMS key used to decrypt the Tink keyset (format: aws-kms://arn:aws:kms:<region>:<account-id>:key/<key-id>)",
			OptType:     types.String,
			ConfigKey:   &c.EncryptionKMSKeyURI,
			FlagDefault: "",
			Required:    true,
		},
		{
			Name:        "encryption-tink-keyset",
			Usage:       "Tink keyset in JSON format to be decrypted",
			OptType:     types.String,
			ConfigKey:   &c.EncryptionTinkKeysetJSON,
			FlagDefault: "",
			Required:    true,
		},
	}
	cmd := &cobra.Command{
		Use:   "decrypt",
		Short: "Decrypt a Tink keyset",
		Long:  "Decrypt a Tink keyset specified in encryption-tink-keyset with the KMS key specified in encryption-kms-key-uri.",
		Run: func(cmd *cobra.Command, args []string) {
			configOpts.Require()
			configOpts.SetValues()
			c.Decrypt()
		},
	}
	configOpts.Init(cmd)

	return cmd
}

func (c *KeysetDecryptCommand) Decrypt() {
	keysetPublic, keysetPrivateCleartext, err := decryptKeyset(c.EncryptionKMSKeyURI, c.EncryptionTinkKeysetJSON)
	if err != nil {
		c.Logger.Errorf("Error decrypting keyset: %v", err)
		return
	}

	c.Logger.Print("Cleartext keyset public:", keysetPublic)
	c.Logger.Print("Cleartext keyset private:", keysetPrivateCleartext)
}

func decryptKeyset(kmsKeyURI, keysetJSON string) (publicCleartext string, privateCleartext string, err error) {
	kmsClient, err := awskms.NewClient(kmsKeyURI)
	if err != nil {
		return "", "", errors.Wrap(err, "initializing AWS KMS client")
	}

	aead, err := kmsClient.GetAEAD(kmsKeyURI)
	if err != nil {
		return "", "", errors.Wrap(err, "getting AEAD primitive from KMS")
	}

	khPriv, err := keyset.Read(keyset.NewJSONReader(strings.NewReader(keysetJSON)), aead)
	if err != nil {
		return "", "", errors.Wrap(err, "getting key handle for keyset private by reading an encrypted keyset")
	}

	keysetPrivateCleartext := strings.Builder{}
	err = insecurecleartextkeyset.Write(khPriv, keyset.NewJSONWriter(&keysetPrivateCleartext))
	if err != nil {
		return "", "", errors.Wrap(err, "writing cleartext keyset private")
	}

	khPub, err := khPriv.Public()
	if err != nil {
		return "", "", errors.Wrap(err, "getting key handle for keyset public")
	}

	keysetPublic := strings.Builder{}
	err = khPub.WriteWithNoSecrets(keyset.NewJSONWriter(&keysetPublic))
	if err != nil {
		return "", "", errors.Wrap(err, "writing cleartext keyset public")
	}

	return keysetPublic.String(), keysetPrivateCleartext.String(), nil
}
