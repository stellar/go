package cmd

import (
	"go/types"
	"strings"

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

type KeysetRotateCommand struct {
	Logger                   *supportlog.Entry
	KeyTemplate              *tinkpb.KeyTemplate
	EncryptionKMSKeyURI      string
	EncryptionTinkKeysetJSON string
}

func (c *KeysetRotateCommand) Command() *cobra.Command {
	configOpts := config.ConfigOptions{
		{
			Name:        "encryption-kms-key-uri",
			Usage:       "URI for a remote KMS key used to decrypt/encrypt Tink keyset during rotation",
			OptType:     types.String,
			ConfigKey:   &c.EncryptionKMSKeyURI,
			FlagDefault: "",
			Required:    false,
		},
		{
			Name:        "encryption-tink-keyset",
			Usage:       "Tink keyset to rotate",
			OptType:     types.String,
			ConfigKey:   &c.EncryptionTinkKeysetJSON,
			FlagDefault: "",
			Required:    true,
		},
	}
	cmd := &cobra.Command{
		Use:   "rotate",
		Short: "Rotate the Tink keyset specified in encryption-tink-keyset by generating a new key, adding it to the keyset, and making it the primary key in the keyset",
		Run: func(cmd *cobra.Command, args []string) {
			configOpts.Require()
			configOpts.SetValues()
			c.Rotate()
		},
	}
	configOpts.Init(cmd)

	return cmd
}

func (c *KeysetRotateCommand) Rotate() {
	keysetPublic, keysetPrivateCleartext, keysetPrivateEncrypted, err := rotateKeyset(c.EncryptionKMSKeyURI, c.EncryptionTinkKeysetJSON, c.KeyTemplate)
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

func rotateKeyset(kmsKeyURI, keysetJSON string, keyTemplate *tinkpb.KeyTemplate) (publicCleartext string, privateCleartext string, privateEncrypted string, err error) {
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

		khPriv, err = keyset.Read(keyset.NewJSONReader(strings.NewReader(keysetJSON)), aead)
		if err != nil {
			return "", "", "", errors.Wrap(err, "getting key handle for keyset private by reading an encrypted keyset")
		}
	} else {
		khPriv, err = insecurecleartextkeyset.Read(keyset.NewJSONReader(strings.NewReader(keysetJSON)))
		if err != nil {
			return "", "", "", errors.Wrap(err, "getting key handle for keyset private by reading a cleartext keyset")
		}
	}

	m := keyset.NewManagerFromHandle(khPriv)
	err = m.Rotate(keyTemplate)
	if err != nil {
		return "", "", "", errors.Wrap(err, "rotating keyset")
	}

	khPriv, err = m.Handle()
	if err != nil {
		return "", "", "", errors.Wrap(err, "creating key handle for the rotated keyset private")
	}

	keysetPrivateEncrypted := strings.Builder{}
	keysetPrivateCleartext := strings.Builder{}
	keysetPublic := strings.Builder{}

	if kmsKeyURI != "" {
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
