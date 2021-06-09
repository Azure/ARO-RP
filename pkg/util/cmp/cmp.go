package cmp

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"crypto/x509"
	"math/big"

	"github.com/google/go-cmp/cmp"

	"github.com/Azure/ARO-RP/pkg/api"
)

// Diff is a wrapper for github.com/google/go-cmp/cmp.Diff with extra options
func Diff(x, y interface{}, opts ...cmp.Option) string {
	newOpts := append(
		opts,
		// FIXME: Remove x509CertComparer after upgrading to a Go version that includes https://github.com/golang/go/issues/28743
		cmp.Comparer(x509CertComparer),
		cmp.Comparer(bigIntComparer),
		cmp.AllowUnexported(api.MissingFields{}),
	)

	return cmp.Diff(x, y, newOpts...)
}

func x509CertComparer(x, y *x509.Certificate) bool {
	if x == nil || y == nil {
		return x == y
	}

	return x.Equal(y)
}

func bigIntComparer(x, y *big.Int) bool {
	if x == nil || y == nil {
		return x == y
	}

	return x.Cmp(y) == 0
}
