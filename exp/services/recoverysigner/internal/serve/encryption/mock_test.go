package encryption

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	awskms "github.com/aws/aws-sdk-go/service/kms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKMS_mock(t *testing.T) {
	kmsProvider := "mock"
	kms, err := NewKMS(kmsProvider, "")
	require.NoError(t, err)
	assert.IsType(t, (*mockAWSKMS)(nil), kms.awsKMSClient)
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

func TestMockAWSKMS_GetPublicKey(t *testing.T) {
	awsKMSClient := &mockAWSKMS{}
	mockKeyId := "mock"
	gpki := &awskms.GetPublicKeyInput{KeyId: aws.String(mockKeyId)}
	gpko, err := awsKMSClient.GetPublicKey(gpki)
	require.NoError(t, err)
	wantPubKey := fmt.Sprintf("key-id-%s", mockKeyId)
	assert.Equal(t, []byte(wantPubKey), gpko.PublicKey)
}
