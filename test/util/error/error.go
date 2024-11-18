package error

import (
	"slices"
	"testing"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// AssertErrorMessage asserts that err.Error() is equal to wantMsg.
func AssertErrorMessage(t *testing.T, err error, wantMsg string) {
	t.Helper()
	if err == nil && wantMsg != "" {
		t.Errorf("did not get an error, but wanted error '%v'", wantMsg)
	}

	if err != nil && err.Error() != wantMsg {
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
