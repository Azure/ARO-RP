package pointerutils

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:fix inline
func ToPtr[T any](t T) *T { return new(t) }

func ToSlicePtr[T any](t []T) []*T {
	x := []*T{}
	for _, i := range t {
		x = append(x, new(i))
	}
	return x
}
