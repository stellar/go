package encryption

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	awskms "github.com/aws/aws-sdk-go/service/kms"
	"github.com/stellar/go/support/errors"
)

type EncryptionContext map[string]*string

type KMS interface {
	Encrypt(msg []byte, ec EncryptionContext) ([]byte, error)
	Decrypt(msg []byte, ec EncryptionContext) ([]byte, error)
}

func NewKMS(kmsProvider, keyID string) (KMS, error) {
	switch kmsProvider {
	case "aws":
		sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
		if err != nil {
			return nil, errors.Wrap(err, "initializing a AWS SDK session")
		}

		kms := AWSKMS{
			awsKMSClient: awskms.New(sess),
			keyID:        keyID,
		}

		return kms, nil

	case "mock":
		return AWSKMS{awsKMSClient: &mockAWSKMS{}}, nil
	}

	return nil, errors.New("KMS_PROVIDER is not set to a valid value, must be 'aws' (default) or 'mock'")
}
