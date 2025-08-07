package encryption

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type AEAD interface {
	Name() string
	Open([]byte) ([]byte, error)
	Seal([]byte) ([]byte, error)
}
