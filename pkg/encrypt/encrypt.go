package encrypt

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/chacha20poly1305"
)

const prefix = "ENC*"

type Cipher interface {
	Decrypt(string) (string, error)
	Encrypt(string) (string, error)
}

var _ Cipher = (*ChaChaCipher)(nil)

type ChaChaCipher struct {
	aead cipher.AEAD
}

func New(key []byte) (*ChaChaCipher, error) {
	i, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	if i != 32 {
		return nil, fmt.Errorf("key length must me 32 byte")
	}
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	return &ChaChaCipher{
		aead: aead,
	}, nil
}

func (c *ChaChaCipher) Decrypt(input string) (string, error) {
	if !strings.HasPrefix(input, prefix) {
		return input, nil
	}

	encoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(input, prefix))
	if err != nil {
		return "", err
	}

	nonce := encoded[:24]
	cipherText := encoded[24:]
	plaintext, err := c.aead.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt or authenticate message: %s", err)
	}

	return string(plaintext), nil
}

func (c *ChaChaCipher) Encrypt(input string) (string, error) {
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	encrypted := c.aead.Seal(nil, nonce, []byte(input), nil)
	cipherText := append(nonce, encrypted...)
	result := base64.StdEncoding.EncodeToString(append(cipherText))
	result = prefix + result

	return result, nil
}
