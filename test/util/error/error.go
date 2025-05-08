package error

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"regexp"
	"slices"
	"testing"
)

type assertOptions int

const trimWhitespace assertOptions = 1

func TrimWhitespace() assertOptions {
	return trimWhitespace
}

// AssertErrorMessage asserts that err.Error() is equal to wantMsg.
func AssertErrorMessage(t *testing.T, err error, wantMsg string, opts ...assertOptions) {
	t.Helper()
	if err == nil && wantMsg != "" {
		t.Errorf("did not get an error, but wanted error '%v'", wantMsg)
	}

	var gotErr string
	wantedErr := wantMsg

	if err != nil {
		gotErr = err.Error()
	}

	for _, i := range opts {
		// trim trailing whitespace in the error message
		if i == trimWhitespace {
			r := regexp.MustCompile(`(?m)[ \t]+(\r?$)`)

			gotErr = r.ReplaceAllString(gotErr, "")
			wantedErr = r.ReplaceAllString(wantedErr, "")
		}
	}

	if err != nil && gotErr != wantedErr {
		t.Errorf("got error '%v', but wanted error '%v'", err, wantMsg)
	}
}

// AssertOneOfErrorMessages asserts that err.Error() is in wantMsgs.
func AssertOneOfErrorMessages(t *testing.T, err error, wantMsgs []string) {
	t.Helper()
	if err == nil && len(wantMsgs) > 0 {
		t.Errorf("did not get an error, but wanted one of these errors: '%v'", wantMsgs)
	}

	if err != nil && !slices.Contains(wantMsgs, err.Error()) {
		t.Errorf("got error '%v', but wanted one of these errors: '%v'", err, wantMsgs)
	}
}
