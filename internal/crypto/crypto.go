// Package crypto provides AES-256-GCM encryption helpers used to protect
// sensitive values (plugin secrets, tokens, passwords) stored in the database.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// encPrefix is prepended to all encrypted values so they can be identified
// and distinguished from legacy plaintext values in the database.
const encPrefix = "enc:v1:"

// Encrypt encrypts plaintext with the given 32-byte AES-256-GCM key and
// returns a prefixed, base64-encoded string safe for database storage.
// A fresh random nonce is generated for each call.
func Encrypt(key []byte, plaintext []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}
	// Seal appends ciphertext+tag to nonce so the nonce is self-contained.
	sealed := gcm.Seal(nonce, nonce, plaintext, nil)
	return encPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt decrypts a value produced by Encrypt. If the value does not carry
// the enc:v1: prefix it is returned as-is — this provides backward
// compatibility with plaintext values written before encryption was enabled.
func Decrypt(key []byte, value string) ([]byte, error) {
	if !strings.HasPrefix(value, encPrefix) {
		// Legacy plaintext — return unchanged.
		return []byte(value), nil
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, encPrefix))
	if err != nil {
		return nil, fmt.Errorf("decoding ciphertext: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}
	return plaintext, nil
}

// IsEncrypted reports whether value was produced by Encrypt.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, encPrefix)
}
