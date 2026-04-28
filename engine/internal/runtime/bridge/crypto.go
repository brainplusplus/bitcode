package bridge

import (
	"github.com/bitcode-framework/bitcode/pkg/security"
)

type cryptoBridge struct {
	encryptor *security.FieldEncryptor
}

func newCryptoBridge(encryptor *security.FieldEncryptor) *cryptoBridge {
	return &cryptoBridge{encryptor: encryptor}
}

func (c *cryptoBridge) Encrypt(plaintext string) (string, error) {
	if c.encryptor == nil {
		return "", NewError(ErrCryptoError, "encryption key not configured")
	}
	result, err := c.encryptor.Encrypt(plaintext)
	if err != nil {
		return "", NewError(ErrCryptoError, err.Error())
	}
	return result, nil
}

func (c *cryptoBridge) Decrypt(ciphertext string) (string, error) {
	if c.encryptor == nil {
		return "", NewError(ErrCryptoError, "encryption key not configured")
	}
	result, err := c.encryptor.Decrypt(ciphertext)
	if err != nil {
		return "", NewError(ErrCryptoError, err.Error())
	}
	return result, nil
}

func (c *cryptoBridge) Hash(value string) (string, error) {
	hash, err := security.HashPassword(value)
	if err != nil {
		return "", NewError(ErrCryptoError, err.Error())
	}
	return hash, nil
}

func (c *cryptoBridge) Verify(value, hash string) (bool, error) {
	return security.CheckPassword(value, hash), nil
}
