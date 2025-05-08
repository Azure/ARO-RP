package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type contextKey int

const (
	ContextKeyLog contextKey = iota
	ContextKeyOriginalPath
	ContextKeyBody
	ContextKeySystemData
)
