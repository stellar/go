package cryptotest

// DecrypterPanic is a decrypter that panics whenever it is called. It is
// intended for use in situations where the decrypter isn't expected to be
// called so as to fail if the decrypter was called to decrypt data. If that
// happens it is a good signal that data that shouldn't be decrypted is being
// decrypted.
type DecrypterPanic struct{}

func (DecrypterPanic) Decrypt(ciphertext, contextInfo []byte) ([]byte, error) {
	panic("decrypter panic")
}
