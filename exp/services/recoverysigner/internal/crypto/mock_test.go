package crypto

import "github.com/google/tink/go/tink"

type mockAEAD struct{}

func (m mockAEAD) Encrypt(plaintext, additionalData []byte) ([]byte, error) {
	return plaintext, nil
}

func (m mockAEAD) Decrypt(ciphertext, additionalData []byte) ([]byte, error) {
	return ciphertext, nil
}

type mockKMSClient struct{}

func (m mockKMSClient) Supported(keyURI string) bool {
	return keyURI != ""
}

func (m mockKMSClient) GetAEAD(keyURI string) (tink.AEAD, error) {
	return mockAEAD{}, nil
}
