package encryption

import (
	"testing"

	awskms "github.com/aws/aws-sdk-go/service/kms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKMS_aws(t *testing.T) {
	kmsProvider := "aws"
	kms, err := NewKMS(kmsProvider, "")
	require.NoError(t, err)
	awsKMS, ok := kms.(AWSKMS)
	assert.True(t, ok)
	assert.IsType(t, (*awskms.KMS)(nil), awsKMS.awsKMSClient)
}

func TestNewKMS_mock(t *testing.T) {
	kmsProvider := "mock"
	kms, err := NewKMS(kmsProvider, "")
	require.NoError(t, err)
	awsKMS, ok := kms.(AWSKMS)
	assert.True(t, ok)
	assert.IsType(t, (*mockAWSKMS)(nil), awsKMS.awsKMSClient)
}

func TestMockAWSKMS_EncryptDecrypt(t *testing.T) {
	awsKMSClient := &mockAWSKMS{}
	ei := &awskms.EncryptInput{Plaintext: []byte("Hello world")}
	eo, err := awsKMSClient.Encrypt(ei)
	require.NoError(t, err)
	di := &awskms.DecryptInput{CiphertextBlob: eo.CiphertextBlob}
	do, err := awsKMSClient.Decrypt(di)
	require.NoError(t, err)
	assert.Equal(t, []byte("Hello world"), do.Plaintext)
}
