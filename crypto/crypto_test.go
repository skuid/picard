package crypto

import (
	"crypto/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetEncryptionKey(t *testing.T) {
	testCases := []struct {
		description string
		giveKey     []byte
		wantErr     string
	}{
		// Happy Path
		{
			"Should set key with no error if 32 bytes",
			[]byte("the-key-has-to-be-32-bytes-long!"),
			"",
		},
		// Sad Path
		{
			"Should return error on short key",
			[]byte("key too short"),
			"encryption keys must be 32 bytes",
		},
		{
			"Should return error on short key",
			[]byte("an extremely long, incredibly verbose, definitely not the right size key"),
			"encryption keys must be 32 bytes",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			err := SetEncryptionKey(tc.giveKey)
			if tc.wantErr != "" {
				assert.Error(t, err)
				assert.EqualError(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.giveKey, encryptionKey)
			}
		})
	}
}

func TestGetEncryptionKey(t *testing.T) {
	testCases := []struct {
		description string
		setKeyPrior bool
		giveKey     []byte
		wantErr     string
	}{
		// Happy Path
		{
			"Should return key with no error after set",
			true,
			[]byte("the-key-has-to-be-32-bytes-long!"),
			"",
		},
		// Sad Path
		{
			"Should generate key with no error",
			false,
			[]byte("the-key-has-to-be-32-bytes-long!"),
			"no encryption key set for picard",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			if tc.setKeyPrior {
				encryptionKey = tc.giveKey
			}
			key, err := GetEncryptionKey()
			if tc.wantErr != "" {
				assert.Error(t, err)
				assert.EqualError(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.giveKey, key)
			}
			encryptionKey = nil
		})
	}
}
func TestGenerateNewEncryptionKey(t *testing.T) {
	testCases := []struct {
		description string
		readerValue string
		shouldError bool
	}{
		// Happy Path
		{
			"Should generate key with no error",
			"the-key-has-to-be-32-bytes-long!",
			false,
		},
		// Sad Path
		{
			"Should return error on any error condition (too few bytes in this case)",
			"y no 32 bytes to spare",
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			oldReader := rand.Reader
			rand.Reader = strings.NewReader(tc.readerValue)
			key, err := GenerateNewEncryptionKey()
			rand.Reader = oldReader
			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, key, []byte(tc.readerValue))
			}
		})
	}
}

func TestEncrypt(t *testing.T) {
	testCases := []struct {
		description string
		plaintext   []byte
		key         []byte
		nonce       string
		wantReturn  []byte
		wantErr     string
	}{
		// Happy Path
		{
			"Should generate key with no error",
			[]byte("some plaintext for encryption"),
			[]byte("the-key-has-to-be-32-bytes-long!"),
			"123412341234",
			[]byte{0x31, 0x32, 0x33, 0x34, 0x31, 0x32, 0x33, 0x34, 0x31, 0x32, 0x33, 0x34, 0x89, 0xb7, 0x60, 0x68, 0x88, 0x29, 0xc2, 0x35, 0xe9, 0x21, 0xb, 0x3a, 0xe3, 0x9b, 0xd9, 0xf1, 0xf5, 0xc7, 0xb, 0xce, 0x67, 0x0, 0xa9, 0xaf, 0xa2, 0x1e, 0xcc, 0x84, 0x5f, 0xbd, 0x6, 0x4f, 0xe6, 0x2c, 0x54, 0xc7, 0xdc, 0x57, 0x4, 0xe2, 0xa4, 0xca, 0x2, 0x2e, 0x5e},
			"",
		},
		{
			"test with different plaintext, key, and nonce",
			[]byte("A different plaintext for test"),
			[]byte("the-key-really-is-32-bytes-long!"),
			"432143214321",
			[]byte{0x34, 0x33, 0x32, 0x31, 0x34, 0x33, 0x32, 0x31, 0x34, 0x33, 0x32, 0x31, 0xb6, 0x47, 0x5f, 0xfe, 0xd4, 0xde, 0x4f, 0x30, 0x6e, 0x70, 0x1, 0x25, 0xdb, 0xe1, 0x45, 0x19, 0x6e, 0xe, 0xd2, 0xc1, 0x39, 0x7, 0xb5, 0xf1, 0xc8, 0x3f, 0x41, 0xd, 0xb4, 0x7a, 0xd, 0xb2, 0xd5, 0x13, 0x33, 0x94, 0x1, 0xda, 0xac, 0x20, 0xe3, 0x32, 0x38, 0xd4, 0xea, 0xfc},
			"",
		},
		// Sad Path
		{
			"Should return error on short key",
			[]byte("some plaintext for encryption"),
			[]byte("short-key"),
			"123412341234",
			[]byte{},
			"crypto/aes: invalid key size 9",
		},
		{
			"Should return error on long key",
			[]byte("some plaintext for encryption"),
			[]byte("an extremely long, incredibly verbose, definitely not the right size key"),
			"123412341234",
			[]byte{},
			"crypto/aes: invalid key size 72",
		},
		{
			"Should return error on bad nonce acquisition",
			[]byte("some plaintext for encryption"),
			[]byte("the-key-has-to-be-32-bytes-long!"),
			"1",
			[]byte{},
			"unexpected EOF",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			oldReader := rand.Reader
			rand.Reader = strings.NewReader(tc.nonce)
			result, err := encrypt(tc.plaintext, tc.key)
			rand.Reader = oldReader
			if tc.wantErr != "" {
				assert.Error(t, err)
				assert.EqualError(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantReturn, result)
			}
		})
	}
}
func TestDecrypt(t *testing.T) {
	testCases := []struct {
		description    string
		encryptedValue []byte
		key            []byte
		wantReturn     []byte
		wantErr        string
	}{
		// Happy Path
		{
			"Should generate key with no error",
			[]byte{0x31, 0x32, 0x33, 0x34, 0x31, 0x32, 0x33, 0x34, 0x31, 0x32, 0x33, 0x34, 0x89, 0xb7, 0x60, 0x68, 0x88, 0x29, 0xc2, 0x35, 0xe9, 0x21, 0xb, 0x3a, 0xe3, 0x9b, 0xd9, 0xf1, 0xf5, 0xc7, 0xb, 0xce, 0x67, 0x0, 0xa9, 0xaf, 0xa2, 0x1e, 0xcc, 0x84, 0x5f, 0xbd, 0x6, 0x4f, 0xe6, 0x2c, 0x54, 0xc7, 0xdc, 0x57, 0x4, 0xe2, 0xa4, 0xca, 0x2, 0x2e, 0x5e},
			[]byte("the-key-has-to-be-32-bytes-long!"),
			[]byte("some plaintext for encryption"),
			"",
		},
		{
			"test with different plaintext, key, and nonce",
			[]byte{0x34, 0x33, 0x32, 0x31, 0x34, 0x33, 0x32, 0x31, 0x34, 0x33, 0x32, 0x31, 0xb6, 0x47, 0x5f, 0xfe, 0xd4, 0xde, 0x4f, 0x30, 0x6e, 0x70, 0x1, 0x25, 0xdb, 0xe1, 0x45, 0x19, 0x6e, 0xe, 0xd2, 0xc1, 0x39, 0x7, 0xb5, 0xf1, 0xc8, 0x3f, 0x41, 0xd, 0xb4, 0x7a, 0xd, 0xb2, 0xd5, 0x13, 0x33, 0x94, 0x1, 0xda, 0xac, 0x20, 0xe3, 0x32, 0x38, 0xd4, 0xea, 0xfc},
			[]byte("the-key-really-is-32-bytes-long!"),
			[]byte("A different plaintext for test"),
			"",
		},
		// Sad Path
		{
			"Should return error on short key",
			[]byte{},
			[]byte("short-key"),
			[]byte("some plaintext for encryption"),
			"crypto/aes: invalid key size 9",
		},
		{
			"Should return error on long key",
			[]byte{},
			[]byte("an extremely long, incredibly verbose, definitely not the right size key"),
			[]byte("some plaintext for encryption"),
			"crypto/aes: invalid key size 72",
		},
		{
			"Should return error on ciphertext too short",
			[]byte{0x34},
			[]byte("the-key-really-is-32-bytes-long!"),
			[]byte{},
			"ciphertext too short",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result, err := decrypt(tc.encryptedValue, tc.key)
			if tc.wantErr != "" {
				assert.Error(t, err)
				assert.EqualError(t, err, tc.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantReturn, result)
			}
		})
	}
}
