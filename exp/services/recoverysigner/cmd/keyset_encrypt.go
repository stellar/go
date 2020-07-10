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

type KeysetEncryptCommand struct {
	Logger                   *supportlog.Entry
	EncryptionKMSKeyURI      string
	EncryptionTinkKeysetJSON string
}

func (c *KeysetEncryptCommand) Command() *cobra.Command {
	configOpts := config.ConfigOptions{
		{
			Name:        "encryption-kms-key-uri",
			Usage:       "URI for a remote KMS key used to encrypt the Tink keyset (format: aws-kms://arn:aws:kms:<region>:<account-id>:key/<key-id>)",
			OptType:     types.String,
			ConfigKey:   &c.EncryptionKMSKeyURI,
			FlagDefault: "",
			Required:    true,
		},
		{
			Name:        "encryption-tink-keyset",
			Usage:       "Tink keyset in JSON format to be encrypted",
			OptType:     types.String,
			ConfigKey:   &c.EncryptionTinkKeysetJSON,
			FlagDefault: "",
			Required:    true,
		},
	}
	cmd := &cobra.Command{
		Use:   "encrypt",
		Short: "Encrypt a Tink keyset",
		Long:  "Encrypt a Tink keyset specified in encryption-tink-keyset with the KMS key specified in encryption-kms-key-uri",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
		},
		Run: func(cmd *cobra.Command, args []string) {
			configOpts.Require()
			configOpts.SetValues()
			c.Encrypt()
		},
	}
	configOpts.Init(cmd)

	return cmd
}

func (c *KeysetEncryptCommand) Encrypt() {
	keysetPublic, keysetPrivateEncrypted, err := encryptKeyset(c.EncryptionKMSKeyURI, c.EncryptionTinkKeysetJSON)
	if err != nil {
		c.Logger.Errorf("Error encrypting keyset: %v", err)
		return
	}

	c.Logger.Print("Cleartext keyset public:", keysetPublic)
	c.Logger.Print("Encrypted keyset private:", keysetPrivateEncrypted)
}

func encryptKeyset(kmsKeyURI, keysetJSON string) (publicCleartext string, privateEncrypted string, err error) {
	kmsClient, err := awskms.NewClient(kmsKeyURI)
	if err != nil {
		return "", "", errors.Wrap(err, "initializing AWS KMS client")
	}

	aead, err := kmsClient.GetAEAD(kmsKeyURI)
	if err != nil {
		return "", "", errors.Wrap(err, "getting AEAD primitive from KMS")
	}

	khPriv, err := insecurecleartextkeyset.Read(keyset.NewJSONReader(strings.NewReader(keysetJSON)))
	if err != nil {
		return "", "", errors.Wrap(err, "getting key handle for keyset private by reading a cleartext keyset")
	}

	keysetPrivateEncrypted := strings.Builder{}
	err = khPriv.Write(keyset.NewJSONWriter(&keysetPrivateEncrypted), aead)
	if err != nil {
		return "", "", errors.Wrap(err, "writing encrypted keyset private")
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

	return keysetPublic.String(), keysetPrivateEncrypted.String(), nil
}
