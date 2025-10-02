package error

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"fmt"
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

// errfmt formats errors.Join() results a little nicer
func errfmt(err error) string {
	u, ok := err.(interface {
		Unwrap() []error
	})
	if ok {
		return fmt.Sprintf("%T(len: %d)", u, len(u.Unwrap()))
	}
	return fmt.Sprintf("%#v", err)
}

// unwrap will return the error tree of a given error starting at a given
// indentation (calling itself for inner-errors until it has the whole tree)
func unwrap(err error, indent string) string {
	gotError := ""
	u, ok := err.(interface {
		Unwrap() []error
	})

	if ok {
		errs := u.Unwrap()
		gotError = gotError + "\n" + indent + "  which contains:"
		for _, e := range errs {
			gotError = gotError + fmt.Sprintf("\n%s    %s", indent, errfmt(e)) + unwrap(e, indent+"    ")
		}
	}

	e := errors.Unwrap(err)
	if e != nil {
		gotError = gotError + fmt.Sprintf("\n%s  which contains:\n    %s%s", indent, indent, errfmt(e)) + unwrap(e, indent+"      ")
	}

	return gotError
}

// AssertErrorMatchesAll verifies that err contains all of the errors in wantError in its tree.
func AssertErrorMatchesAll(t *testing.T, err error, wantError []error) {
	t.Helper()
	notFound := []error{}

	if err == nil && len(wantError) == 0 {
		return
	} else if err != nil && len(wantError) == 0 {
		t.Errorf("got unexpected error '%v'", err)
	} else if err == nil && len(wantError) != 0 {
		t.Errorf("got unexpected SUCCESS instead of errors '%v'", wantError)
	} else {
		for _, wanted := range wantError {
			errorMatched := false
			if errors.Is(err, wanted) {
				errorMatched = true
			} else {
				// check the content in case it's just plain error strings
				if err.Error() == wanted.Error() {
					errorMatched = true
				}
			}

			if !errorMatched {
				notFound = append(notFound, wanted)
			}
		}
	}

	if len(notFound) > 0 {
		errt := ""
		for _, w := range notFound {
			errt = errt + fmt.Sprintf("\n  %s", errfmt(w))
		}

		t.Errorf("error mismatch\ngot error:\n  %s%s\n\ncould not find the errors:%s", errfmt(err), unwrap(err, "  "), errt)
	}
}
