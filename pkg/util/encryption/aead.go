package encryption

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "errors"

// ErrKeyMismatch is wrapped by the error from Open when the input fails
// authentication: it was sealed under a key other than the one held, or has
// been altered since. Test for it with errors.Is.
//
// A multi refreshes its keys only when an open fails with ErrKeyMismatch, or
// when it holds no keys at all, since no other failure could be cured by a
// different key. An AEAD which does not wrap its authentication failures in
// ErrKeyMismatch therefore never causes a refresh.
var ErrKeyMismatch = errors.New("encryption: key mismatch")

type AEAD interface {
	Open([]byte) ([]byte, error)
	Seal([]byte) ([]byte, error)
}
