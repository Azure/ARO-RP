package encryption

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"github.com/codahale/etm"
)

type aes256Sha512 struct {
	aead       cipher.AEAD
	randReader io.Reader
}

var _ AEAD = (*aes256Sha512)(nil)

func NewAES256SHA512(ctx context.Context, key []byte) (AEAD, error) {
	aead, err := etm.NewAES256SHA512(key)
	if err != nil {
		return nil, err
	}

	return &aes256Sha512{
		aead:       aead,
		randReader: rand.Reader,
	}, nil
}

func (c *aes256Sha512) Open(input []byte) ([]byte, error) {
	if len(input) < 32 {
		return nil, fmt.Errorf("encrypted value too short")
	}

	return c.aead.Open(nil, nil, input, nil)
}

func (c *aes256Sha512) Seal(input []byte) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())

	_, err := io.ReadFull(c.randReader, nonce)
	if err != nil {
		return nil, err
	}

	return c.aead.Seal(nil, nonce, input, nil), nil
}
