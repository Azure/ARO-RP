package encryption

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/Azure/ARO-RP/pkg/env"
)

var _ Cipher = (*aeadCipher)(nil)

type Cipher interface {
	Decrypt([]byte) ([]byte, error)
	Encrypt([]byte) ([]byte, error)
}

type aeadCipher struct {
	aead     cipher.AEAD
	randRead func([]byte) (int, error)
}

func NewXChaCha20Poly1305(ctx context.Context, _env env.Interface, secretName string) (Cipher, error) {
	key, err := _env.ServiceKeyvault().GetBase64Secret(ctx, secretName)
	if err != nil {
		return nil, err
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	return &aeadCipher{
		aead:     aead,
		randRead: rand.Read,
	}, nil
}

func (c *aeadCipher) Decrypt(input []byte) ([]byte, error) {
	if len(input) < chacha20poly1305.NonceSizeX {
		return nil, fmt.Errorf("encrypted value too short")
	}

	nonce := input[:chacha20poly1305.NonceSizeX]
	data := input[chacha20poly1305.NonceSizeX:]

	return c.aead.Open(nil, nonce, data, nil)
}

func (c *aeadCipher) Encrypt(input []byte) ([]byte, error) {
	nonce := make([]byte, chacha20poly1305.NonceSizeX)

	n, err := c.randRead(nonce)
	if err != nil {
		return nil, err
	}

	if n != chacha20poly1305.NonceSizeX {
		return nil, fmt.Errorf("rand.Read returned %d bytes, expected %d", n, chacha20poly1305.NonceSizeX)
	}

	return append(nonce, c.aead.Seal(nil, nonce, input, nil)...), nil
}
