package encryption

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/Azure/ARO-RP/pkg/env"
)

// encryptionSecretName must match key name in the service keyvault
const (
	encryptionSecretName = "encryption-key"
	Prefix               = "ENC*"
)

var (
	_        Cipher = (*aeadCipher)(nil)
	RandRead        = rand.Read
)

type Cipher interface {
	Decrypt(string) (string, error)
	Encrypt(string) (string, error)
}

type aeadCipher struct {
	aead cipher.AEAD
}

func NewCipher(ctx context.Context, env env.Interface) (Cipher, error) {
	keybase64, err := env.GetSecret(ctx, encryptionSecretName)
	if err != nil {
		return nil, err
	}

	key := make([]byte, base64.StdEncoding.DecodedLen(len(keybase64)))
	n, err := base64.StdEncoding.Decode(key, keybase64)
	if err != nil {
		return nil, err
	}

	if n < 32 {
		return nil, fmt.Errorf("chacha20poly1305: bad key length")
	}
	key = key[:32]

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	return &aeadCipher{
		aead: aead,
	}, nil
}

// Decrypt decrypts input
func (c *aeadCipher) Decrypt(input string) (string, error) {
	if !strings.HasPrefix(input, Prefix) {
		return input, nil
	}
	input = input[len(Prefix):]

	r := make([]byte, base64.StdEncoding.DecodedLen(len(input)))
	r, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return "", err
	}

	if len(r) >= 24 {
		nonce := r[0:24]
		data := r[24:]
		output, err := c.aead.Open(nil, nonce, data, nil)
		if err != nil {
			return "", err
		}
		return string(output), nil
	}
	return "", fmt.Errorf("error while decrypting message")
}

// Encrypt encrypts input using 24 byte nonce
func (c *aeadCipher) Encrypt(input string) (string, error) {
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	_, err := RandRead(nonce)
	if err != nil {
		return "", err
	}
	encrypted := c.aead.Seal(nil, nonce, []byte(input), nil)

	var encryptedFinal []byte
	encryptedFinal = append(encryptedFinal, nonce...)
	encryptedFinal = append(encryptedFinal, encrypted...)

	encryptedBase64 := make([]byte, base64.StdEncoding.EncodedLen(len(encryptedFinal)))
	base64.StdEncoding.Encode(encryptedBase64, encryptedFinal)

	// return prefix+base64(nonce+encryptedFinal)
	var result []byte
	result = append(result, Prefix...)
	result = append(result, encryptedBase64...)
	return string(result), nil
}
