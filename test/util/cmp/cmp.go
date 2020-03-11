package cmp

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	gocmp "github.com/google/go-cmp/cmp"

	"github.com/Azure/ARO-RP/pkg/api"
)

// Diff is a wrapper for github.com/google/go-cmp/cmp.Diff with extra options
func Diff(x, y interface{}, opts ...gocmp.Option) string {
	newOpts := append(
		opts,
		gocmp.AllowUnexported(api.MissingFields{}),
	)

	return gocmp.Diff(x, y, newOpts...)
}
