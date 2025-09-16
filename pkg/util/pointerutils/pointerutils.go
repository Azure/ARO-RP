package pointerutils

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func ToPtr[T any](t T) *T { return &t }

func ToSlicePtr[T any](t []T) []*T {
	x := []*T{}
	for _, i := range t {
		x = append(x, ToPtr(i))
	}
	return x
}
