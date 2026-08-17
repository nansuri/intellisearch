package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// EncryptSecret seals plaintext with AES-GCM using the key derived from the
// configured encryption key. The returned string is base64(nonce || ciphertext).
func EncryptSecret(plaintext string, key []byte) (string, error) {
	block, err := aes.NewCipher(normalizeKey(key))
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nil, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(append(nonce, sealed...)), nil
}

// DecryptSecret reverses EncryptSecret.
func DecryptSecret(encoded string, key []byte) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(normalizeKey(key))
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < aead.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, sealed := raw[:aead.NonceSize()], raw[aead.NonceSize():]
	plain, err := aead.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// normalizeKey derives a 32-byte key from whatever length the env value is.
func normalizeKey(key []byte) []byte {
	derived := make([]byte, 32)
	if len(key) == 0 {
		return derived
	}
	copy(derived, key)
	if len(key) < 32 {
		for i := len(key); i < 32; i++ {
			derived[i] = key[i%len(key)]
		}
	}
	return derived
}
