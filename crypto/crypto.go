// Package crypto lets picard column values to be encrypted and dexrypted with an encryption key
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

var encryptionKey []byte

func SetEncryptionKey(key []byte) error {
	if len(key) != 32 {
		return errors.New("encryption keys must be 32 bytes")
	}
	encryptionKey = key
	return nil
}

func GetEncryptionKey() ([]byte, error) {
	if encryptionKey == nil {
		return nil, errors.New("no encryption key set for picard")
	}
	return encryptionKey, nil
}

func GenerateNewEncryptionKey() ([]byte, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func EncryptBytes(v []byte) ([]byte, error) {
	if encryptionKey == nil {
		return nil, errors.New("no encryption key set for picard")
	}
	return encrypt(v, encryptionKey)
}

func encrypt(plaintext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func DecryptBytes(v []byte) ([]byte, error) {
	if encryptionKey == nil {
		return nil, errors.New("no encryption key set for picard")
	}
	return decrypt(v, encryptionKey)
}

func decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
