package pointerutils

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

func ToPtr[T any](t T) *T { return &t }
