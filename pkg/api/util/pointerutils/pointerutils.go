package pointerutils

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

//go:fix inline
func ToPtr[T any](t T) *T { return new(t) }
