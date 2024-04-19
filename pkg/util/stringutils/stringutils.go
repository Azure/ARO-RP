package stringutils

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"regexp"
	"strings"
)

// LastTokenByte splits s on sep and returns the last token
func LastTokenByte(s string, sep byte) string {
	return s[strings.LastIndexByte(s, sep)+1:]
}

func Contains(list []string, value string) bool {
	for _, e := range list {
		if value == e {
			return true
		}
	}
	return false
}

// returns true if a string is formatted like an Azure resource ID
// otherwise, returns false
func IsResourceIDFormatted(id string) bool {
	resourceIDPattern := "/subscriptions/.*/resourceGroups/.*/providers/.*/.*/.*"
	// the regexp is statically defined and hard-coded, we shouldn't need to check for errors
	match, _ := regexp.MatchString(resourceIDPattern, id)

	return match
}
