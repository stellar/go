package cmd

import (
	"go/types"
	"strings"

	"github.com/google/tink/go/hybrid"
	"github.com/google/tink/go/insecurecleartextkeyset"
	"github.com/google/tink/go/integration/awskms"
	"github.com/google/tink/go/keyset"
	tinkpb "github.com/google/tink/go/proto/tink_go_proto"
	"github.com/google/tink/go/tink"
	"github.com/spf13/cobra"
	"github.com/stellar/go/support/config"
	"github.com/stellar/go/support/errors"
	supportlog "github.com/stellar/go/support/log"
)

type KeysetCommand struct {
	Logger               *supportlog.Entry
	EncryptionKMSKeyURI  string
	EncryptionTinkKeyset string
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
		{
			Name:        "encryption-tink-keyset",
			Usage:       "Existing Tink keyset to rotate",
			OptType:     types.String,
			ConfigKey:   &c.EncryptionTinkKeyset,
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
	rotateCmd := &cobra.Command{
		Use:   "rotate",
		Short: "Rotate the Tink keyset specified in encryption-tink-keyset by generating a new key, adding it to the keyset, and making it the primary key in the keyset",
		Run: func(_ *cobra.Command, _ []string) {
			c.Rotate()
		},
	}

	cmd.AddCommand(createCmd)
	cmd.AddCommand(rotateCmd)

	return cmd
}

func (c *KeysetCommand) keyTemplate() *tinkpb.KeyTemplate {
	return hybrid.ECIESHKDFAES128GCMKeyTemplate()
}

func (c *KeysetCommand) Create() {
	keysetPublic, keysetPrivateCleartext, keysetPrivateEncrypted, err := createKeyset(c.EncryptionKMSKeyURI, c.keyTemplate())
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

func (c *KeysetCommand) Rotate() {
	keysetPublic, keysetPrivateCleartext, keysetPrivateEncrypted, err := rotateKeyset(c.EncryptionKMSKeyURI, c.EncryptionTinkKeyset, c.keyTemplate())
	if err != nil {
		c.Logger.Errorf("Error rotating keyset: %v", err)
		return
	}

	c.Logger.Print("Cleartext keyset public:", keysetPublic)
	c.Logger.Print("Cleartext keyset private:", keysetPrivateCleartext)

	if keysetPrivateEncrypted != "" {
		c.Logger.Print("Encrypted keyset private:", keysetPrivateEncrypted)
	}
}

func rotateKeyset(kmsKeyURI, currentTinkKeyset string, keyTemplate *tinkpb.KeyTemplate) (publicCleartext string, privateCleartext string, privateEncrypted string, err error) {
	var (
		khPriv *keyset.Handle
		aead   tink.AEAD
	)

	if kmsKeyURI != "" {
		kmsClient, kmsErr := awskms.NewClient(kmsKeyURI)
		if kmsErr != nil {
			return "", "", "", errors.Wrap(kmsErr, "initializing AWS KMS client")
		}

		aead, kmsErr = kmsClient.GetAEAD(kmsKeyURI)
		if kmsErr != nil {
			return "", "", "", errors.Wrap(kmsErr, "getting AEAD primitive from KMS")
		}

		khPriv, err = keyset.Read(keyset.NewJSONReader(strings.NewReader(currentTinkKeyset)), aead)
		if err != nil {
			return "", "", "", errors.Wrap(err, "reading encrypted keyset")
		}
	} else {
		khPriv, err = insecurecleartextkeyset.Read(keyset.NewJSONReader(strings.NewReader(currentTinkKeyset)))
		if err != nil {
			return "", "", "", errors.Wrap(err, "getting key handle for private key")
		}
	}

	m := keyset.NewManagerFromHandle(khPriv)
	err = m.Rotate(keyTemplate)
	if err != nil {
		return "", "", "", errors.Wrap(err, "rotating keyset")
	}

	khPriv, err = m.Handle()
	if err != nil {
		return "", "", "", errors.Wrap(err, "creating handle for the new keyset")
	}

	keysetPrivateEncrypted := strings.Builder{}
	keysetPrivateCleartext := strings.Builder{}
	keysetPublic := strings.Builder{}

	if kmsKeyURI != "" {
		err = khPriv.Write(keyset.NewJSONWriter(&keysetPrivateEncrypted), aead)
		if err != nil {
			return "", "", "", errors.Wrap(err, "writing encrypted keyset containing private keys")
		}
	}

	err = insecurecleartextkeyset.Write(khPriv, keyset.NewJSONWriter(&keysetPrivateCleartext))
	if err != nil {
		return "", "", "", errors.Wrap(err, "writing cleartext keyset containing private keys")
	}

	khPub, err := khPriv.Public()
	if err != nil {
		return "", "", "", errors.Wrap(err, "getting keyhandle for public keys")
	}

	err = khPub.WriteWithNoSecrets(keyset.NewJSONWriter(&keysetPublic))
	if err != nil {
		return "", "", "", errors.Wrap(err, "writing cleartext keyset containing public keys")
	}

	return keysetPublic.String(), keysetPrivateCleartext.String(), keysetPrivateEncrypted.String(), nil
}
