package encryption

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "errors"

// ErrKeyMismatch is returned by Open when the ciphertext cannot be
// authenticated by any key the caller holds.
var ErrKeyMismatch = errors.New("encryption: key mismatch")

type AEAD interface {
	Open([]byte) ([]byte, error)
	Seal([]byte) ([]byte, error)
}
