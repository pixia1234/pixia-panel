package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

const (
	gcmIVLen  = 12
	gcmTagLen = 16
)

func keyFromSecret(secret string) []byte {
	h := sha256.Sum256([]byte(secret))
	return h[:]
}

func DecryptString(secret, encrypted string) (string, error) {
	plain, err := Decrypt(secret, encrypted)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func Decrypt(secret, encrypted string) ([]byte, error) {
	if secret == "" {
		return nil, errors.New("secret required")
	}
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, err
	}
	if len(data) < gcmIVLen+gcmTagLen {
		return nil, errors.New("ciphertext too short")
	}

	iv := data[:gcmIVLen]
	ciphertext := data[gcmIVLen:]

	block, err := aes.NewCipher(keyFromSecret(secret))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, iv, ciphertext, nil)
}

func EncryptString(secret, plain string) (string, error) {
	out, err := Encrypt(secret, []byte(plain))
	if err != nil {
		return "", err
	}
	return out, nil
}

func Encrypt(secret string, plain []byte) (string, error) {
	if secret == "" {
		return "", errors.New("secret required")
	}
	if len(plain) == 0 {
		return "", errors.New("plaintext required")
	}

	iv := make([]byte, gcmIVLen)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyFromSecret(secret))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nil, iv, plain, nil)
	payload := append(iv, ciphertext...)
	return base64.StdEncoding.EncodeToString(payload), nil
}
