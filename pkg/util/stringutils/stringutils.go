package stringutils

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "strings"

// LastTokenByte splits s on sep and returns the last token
func LastTokenByte(s string, sep byte) string {
	return s[strings.LastIndexByte(s, sep)+1:]
}
