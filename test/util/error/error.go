package error

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
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

func AssertErrorIs(t *testing.T, err error, wantError error) {
	t.Helper()

	if err != nil && wantError == nil {
		t.Errorf("got unexpected error '%v'", err)
	} else if err == nil && wantError != nil {
		t.Errorf("got unexpected SUCCESS instead of error '%v'", wantError)
	} else {
		if !errors.Is(err, wantError) {
			// check the content in case it's just plain error strings
			if err.Error() != wantError.Error() {
				t.Errorf("got error:\n'%v'\n\nwanted error:\n '%v'", err, wantError)
			}
		}
	}
}
