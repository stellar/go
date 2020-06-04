package encryption

import (
	"crypto"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	awskms "github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/pkg/errors"
)

// KMS implements crypto.Decrypter.
var _ crypto.Decrypter = (*KMS)(nil)

type KMS struct {
	// AWS KMS
	awsKMSClient kmsiface.KMSAPI

	// Google Cloud KMS (if we decide to support it in the future)
	// googleKMSClient ...

	// shared attribute
	keyID     string
	publicKey crypto.PublicKey
}

func NewKMS(kmsProvider, keyID string) (*KMS, error) {
	switch kmsProvider {
	case "aws":
		sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
		if err != nil {
			return nil, errors.Wrap(err, "initializing a AWS SDK session")
		}

		kms := KMS{
			awsKMSClient: awskms.New(sess),
			keyID:        keyID,
		}

		gpki := &awskms.GetPublicKeyInput{KeyId: aws.String(kms.keyID)}
		gpko, err := kms.awsKMSClient.GetPublicKey(gpki)
		if err != nil {
			return nil, errors.Wrap(err, "getting public key from AWS KMS")
		}

		pk, err := parsePublicKey(gpko.PublicKey)
		if err != nil {
			return nil, errors.Wrap(err, "parsing public key")
		}
		kms.publicKey = pk
		return &kms, nil

	case "mock":
		return &KMS{awsKMSClient: &mockAWSKMS{}}, nil
	}

	return nil, errors.New("KMS_PROVIDER is not set to a valid value, must be 'aws' (default) or 'mock'")
}

func (k *KMS) Decrypt(_ io.Reader, msg []byte, _ crypto.DecrypterOpts) (plaintext []byte, err error) {
	if k.awsKMSClient != nil {
		di := &awskms.DecryptInput{CiphertextBlob: msg}
		do, err := k.awsKMSClient.Decrypt(di)
		if err != nil {
			return nil, errors.Wrap(err, "decrypting message")
		}

		return do.Plaintext, nil
	}

	// If we decide to support Google Cloud KMS:
	// if k.googleKMSClient != nil {
	// 	resp, err := k.googleKMSClient.AsymmetricDecrypt(...)
	// }

	return nil, errors.New("No KMS client is configured")
}

func (k *KMS) Public() crypto.PublicKey {
	return k.publicKey
}
