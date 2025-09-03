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
	if wantError == nil {
		AssertErrorMatchesAll(t, err, []error{})
		return
	}
	AssertErrorMatchesAll(t, err, []error{wantError})
}

// AssertErrorMatchesAll verifies that err contains all of the errors in wantError in its tree.
func AssertErrorMatchesAll(t *testing.T, err error, wantError []error) {
	t.Helper()

	if err == nil && len(wantError) == 0 {
		return
	} else if err != nil && len(wantError) == 0 {
		t.Errorf("got unexpected error '%v'", err)
	} else if err == nil && len(wantError) != 0 {
		t.Errorf("got unexpected SUCCESS instead of errors '%v'", wantError)
	} else {
		errorMatched := false
		for _, wanted := range wantError {
			if !errors.Is(err, wanted) {
				// check the content in case it's just plain error strings
				if err.Error() == wanted.Error() {
					errorMatched = true
				}
			} else {
				errorMatched = true
			}
		}
		if !errorMatched {
			t.Errorf("got error:\n'%v'\n\nwanted one of errors:\n", err)
			for _, w := range wantError {
				t.Error(w)
			}
		}
	}
}
