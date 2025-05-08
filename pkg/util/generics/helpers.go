package generics

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func ConcatMultipleSlices[T any](slices ...[]T) []T {
	result := []T{}

	for _, s := range slices {
		result = append(result, s...)
	}

	return result
}
