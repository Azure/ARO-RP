package stringutils

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"slices"
	"strings"
)

// LastTokenByte splits s on sep and returns the last token
func LastTokenByte(s string, sep byte) string {
	return s[strings.LastIndexByte(s, sep)+1:]
}

func Contains(list []string, value string) bool {
	return slices.Contains(list, value)
}

func GroupsIntersect(as, bs []string) (gs []string) {
	for _, a := range as {
		if slices.Contains(bs, a) {
			gs = append(gs, a)
		}
	}

	return gs
}

func IndentLines(t string, indent string) string {
	out := &strings.Builder{}
	for l := range strings.Lines(t) {
		out.WriteString(indent)
		out.WriteString(l)
	}
	return out.String()
}

func GroupsUnion(as, bs []string) (gs []string) {
	match := map[string]struct{}{}

	for _, a := range as {
		match[a] = struct{}{}
	}
	for _, b := range bs {
		match[b] = struct{}{}
	}

	for g := range match {
		gs = append(gs, g)
	}
	return gs
}
