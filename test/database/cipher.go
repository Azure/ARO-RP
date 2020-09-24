package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

var fakeCode []byte = []byte{'F', 'A', 'K', 'E'}

type fakeCipher struct {
}

func (c fakeCipher) Decrypt(in []byte) ([]byte, error) {
	return in[4:], nil
}

func (c fakeCipher) Encrypt(in []byte) ([]byte, error) {
	out := make([]byte, 4+len(in))
	copy(out, fakeCode)
	copy(out[4:], in)
	return out, nil
}

func NewFakeCipher() *fakeCipher {
	return &fakeCipher{}
}
