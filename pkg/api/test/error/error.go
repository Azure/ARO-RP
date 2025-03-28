package error

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
)

// AssertErrorMessage asserts that err.Error() is equal to wantMsg.
func AssertErrorMessage(t *testing.T, err error, wantMsg string) {
	t.Helper()
	if err == nil && wantMsg != "" {
		t.Errorf("did not get an error, but wanted error '%v'", wantMsg)
	}

	var gotErr string
	wantedErr := wantMsg

	if err != nil {
		gotErr = err.Error()
	}

	if err != nil && gotErr != wantedErr {
		t.Errorf("got error '%v', but wanted error '%v'", err, wantMsg)
	}
}
