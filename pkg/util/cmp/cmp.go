package cmp

import (
	"crypto/x509"
	"math/big"

	gocmp "github.com/google/go-cmp/cmp"
)

// Diff is a wrapper for github.com/google/go-cmp/cmp.Diff with extra options
func Diff(x, y interface{}, opts ...gocmp.Option) string {
	newOpts := append(
		opts,
		// FIXME: Remove x509CertComparer after upgrading to a Go version that includes https://github.com/golang/go/issues/28743
		gocmp.Comparer(x509CertComparer),
		gocmp.Comparer(bigIntComparer),
	)

	return gocmp.Diff(x, y, newOpts...)
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
