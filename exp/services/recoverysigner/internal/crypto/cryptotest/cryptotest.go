package cryptotest

type EncrypterFunc func(plaintext, contextInfo []byte) ([]byte, error)

func (f EncrypterFunc) Encrypt(plaintext, contextInfo []byte) ([]byte, error) {
	return f(plaintext, contextInfo)
}
