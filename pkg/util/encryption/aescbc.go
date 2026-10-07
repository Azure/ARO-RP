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

// aes256Sha512MinSize is the length of the shortest ciphertext Seal produces:
// a 16-byte IV, one 16-byte block of padded plaintext, and a 32-byte tag. An
// input any shorter cannot have been sealed under any key, and is reported as
// malformed rather than as a key mismatch.
//
// It is not NonceSize()+Overhead(), as it is for XChaCha20-Poly1305. etm's
// Overhead() is an upper bound which already counts the IV, and also counts a
// length field that is never written, so that sum is 88, and would reject
// every valid ciphertext of a plaintext up to 31 bytes long.
const aes256Sha512MinSize = 16 + 16 + 32

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
	if len(input) < aes256Sha512MinSize {
		return nil, fmt.Errorf("encrypted value too short")
	}

	b, err := c.aead.Open(nil, nil, input, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrKeyMismatch, err)
	}

	return b, nil
}

func (c *aes256Sha512) Seal(input []byte) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())

	_, err := io.ReadFull(c.randReader, nonce)
	if err != nil {
		return nil, err
	}

	return c.aead.Seal(nil, nonce, input, nil), nil
}
