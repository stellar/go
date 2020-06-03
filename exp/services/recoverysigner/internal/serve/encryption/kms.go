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

	// Google Cloud KMS (support in the future)

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

	return nil, errors.New("AWSKMSClient is not set")
}

func (k *KMS) Public() crypto.PublicKey {
	return k.publicKey
}

type mockAWSKMS struct {
	kmsiface.KMSAPI
}

func (m *mockAWSKMS) Encrypt(ei *awskms.EncryptInput) (*awskms.EncryptOutput, error) {
	return &awskms.EncryptOutput{CiphertextBlob: ei.Plaintext}, nil
}

func (m *mockAWSKMS) Decrypt(di *awskms.DecryptInput) (*awskms.DecryptOutput, error) {
	return &awskms.DecryptOutput{Plaintext: di.CiphertextBlob}, nil
}
