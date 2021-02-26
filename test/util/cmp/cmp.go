package cmp

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/google/go-cmp/cmp"

	"github.com/Azure/ARO-RP/pkg/api"
)

// Diff is a wrapper for github.com/google/go-cmp/cmp.Diff with extra options
func Diff(x, y interface{}, opts ...cmp.Option) string {
	newOpts := append(
		opts,
		cmp.AllowUnexported(api.MissingFields{}),
	)

	return cmp.Diff(x, y, newOpts...)
}
