package encryption

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	awskms "github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
)

type mockAWSKMS struct {
	kmsiface.KMSAPI
}

func (m *mockAWSKMS) Encrypt(ei *awskms.EncryptInput) (*awskms.EncryptOutput, error) {
	return &awskms.EncryptOutput{CiphertextBlob: ei.Plaintext}, nil
}

func (m *mockAWSKMS) Decrypt(di *awskms.DecryptInput) (*awskms.DecryptOutput, error) {
	return &awskms.DecryptOutput{Plaintext: di.CiphertextBlob}, nil
}

func (m *mockAWSKMS) GetPublicKey(gpki *kms.GetPublicKeyInput) (*kms.GetPublicKeyOutput, error) {
	return &awskms.GetPublicKeyOutput{PublicKey: []byte(fmt.Sprintf("key-id-%s", aws.StringValue(gpki.KeyId)))}, nil
}
