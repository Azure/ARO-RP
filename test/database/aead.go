package database

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

var fakeCode []byte = []byte{'F', 'A', 'K', 'E'}

type fakeAEAD struct {
	secretVersion string
}

func (fakeAEAD) Open(in []byte) ([]byte, error) {
	return in[4:], nil
}

func (f fakeAEAD) Seal(in []byte) ([]byte, error) {
	return append(fakeCode, in...), nil
}

func NewFakeAEAD(secretVersion string) *fakeAEAD {
	return &fakeAEAD{
		secretVersion: secretVersion,
	}
}

func (f fakeAEAD) SealSecretVersion() string {
	return f.secretVersion
}
