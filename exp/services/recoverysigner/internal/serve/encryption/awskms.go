package encryption

import (
	"github.com/aws/aws-sdk-go/aws"
	awskms "github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/stellar/go/support/errors"
)

type AWSKMS struct {
	awsKMSClient kmsiface.KMSAPI
	keyID        string
}

func (ak AWSKMS) Encrypt(msg []byte, ec EncryptionContext) ([]byte, error) {
	ei := &awskms.EncryptInput{
		Plaintext:         msg,
		EncryptionContext: ec,
		KeyId:             aws.String(ak.keyID),
	}
	eo, err := ak.awsKMSClient.Encrypt(ei)
	if err != nil {
		return nil, errors.Wrap(err, "encrypting message")
	}

	return eo.CiphertextBlob, nil
}

func (ak AWSKMS) Decrypt(msg []byte, ec EncryptionContext) ([]byte, error) {
	di := &awskms.DecryptInput{
		CiphertextBlob:    msg,
		EncryptionContext: ec,
		KeyId:             aws.String(ak.keyID),
	}
	do, err := ak.awsKMSClient.Decrypt(di)
	if err != nil {
		return nil, errors.Wrap(err, "decrypting message")
	}

	return do.Plaintext, nil
}
