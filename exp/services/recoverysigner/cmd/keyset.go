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
	Logger                   *supportlog.Entry
	EncryptionKMSKeyURI      string
	EncryptionTinkKeysetJSON string
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
			Usage:       "Tink keyset to rotate/encrypt/decrypt",
			OptType:     types.String,
			ConfigKey:   &c.EncryptionTinkKeysetJSON,
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
	decryptCmd := &cobra.Command{
		Use:   "decrypt",
		Short: "Decrypt the Tink keyset specified in encryption-tink-keyset with the KMS key specified in encryption-kms-key-uri",
		Run: func(_ *cobra.Command, _ []string) {
			c.Decrypt()
		},
	}
	encryptCmd := &cobra.Command{
		Use:   "encrypt",
		Short: "Encrypt the Tink keyset specified in encryption-tink-keyset with the KMS key specified in encryption-kms-key-uri",
		Run: func(_ *cobra.Command, _ []string) {
			c.Encrypt()
		},
	}

	cmd.AddCommand(createCmd)
	cmd.AddCommand(rotateCmd)
	cmd.AddCommand(decryptCmd)
	cmd.AddCommand(encryptCmd)

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

func (c *KeysetCommand) Rotate() {
	keysetPublic, keysetPrivateCleartext, keysetPrivateEncrypted, err := rotateKeyset(c.EncryptionKMSKeyURI, c.EncryptionTinkKeysetJSON, c.keyTemplate())
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

var errNoKMSKeyURI = errors.New("KMS Key URI is not configured")

func (c *KeysetCommand) Decrypt() {
	keysetPublic, keysetPrivateCleartext, err := decryptKeyset(c.EncryptionKMSKeyURI, c.EncryptionTinkKeysetJSON)
	if err != nil {
		c.Logger.Errorf("Error decrypting keyset: %v", err)
		return
	}

	c.Logger.Print("Cleartext keyset public:", keysetPublic)
	c.Logger.Print("Cleartext keyset private:", keysetPrivateCleartext)
}

func decryptKeyset(kmsKeyURI, keysetJSON string) (publicCleartext string, privateCleartext string, err error) {
	if kmsKeyURI == "" {
		return "", "", errNoKMSKeyURI
	}

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

func (c *KeysetCommand) Encrypt() {
	keysetPublic, keysetPrivateEncrypted, err := encryptKeyset(c.EncryptionKMSKeyURI, c.EncryptionTinkKeysetJSON)
	if err != nil {
		c.Logger.Errorf("Error encrypting keyset: %v", err)
		return
	}

	c.Logger.Print("Cleartext keyset public:", keysetPublic)
	c.Logger.Print("Encrypted keyset private:", keysetPrivateEncrypted)
}

func encryptKeyset(kmsKeyURI, keysetJSON string) (publicCleartext string, privateEncrypted string, err error) {
	if kmsKeyURI == "" {
		return "", "", errNoKMSKeyURI
	}

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
