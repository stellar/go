package cryptotest

type DecrypterFunc func(ciphertext, contextInfo []byte) ([]byte, error)

func (f DecrypterFunc) Decrypt(ciphertext, contextInfo []byte) ([]byte, error) {
	return f(ciphertext, contextInfo)
}
