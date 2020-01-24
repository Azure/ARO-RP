package bucket

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/rand"
	"math/big"
)

const (
	Buckets = 256
)

type Allocator interface {
	Allocate() (int, error)
}

type Random struct{}

func (Random) Allocate() (int, error) {
	bucket, err := rand.Int(rand.Reader, big.NewInt(Buckets))
	if err != nil {
		return 0, err
	}

	return int(bucket.Int64()), nil
}

type Fixed int

func (f Fixed) Allocate() (int, error) {
	return int(f), nil
}
