package encryption

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
)

type xChaCha20Poly1305 struct {
	aead       cipher.AEAD
	randReader io.Reader
}

var _ AEAD = (*xChaCha20Poly1305)(nil)

func NewXChaCha20Poly1305(ctx context.Context, key []byte) (AEAD, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	return &xChaCha20Poly1305{
		aead:       aead,
		randReader: rand.Reader,
	}, nil
}

func (c *xChaCha20Poly1305) Open(input []byte) ([]byte, error) {
	if len(input) < c.aead.NonceSize() {
		return nil, fmt.Errorf("encrypted value too short")
	}

	nonce := input[:c.aead.NonceSize()]
	data := input[c.aead.NonceSize():]

	return c.aead.Open(nil, nonce, data, nil)
}

func (c *xChaCha20Poly1305) Seal(input []byte) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())

	_, err := io.ReadFull(c.randReader, nonce)
	if err != nil {
		return nil, err
	}

	return append(nonce, c.aead.Seal(nil, nonce, input, nil)...), nil
}
