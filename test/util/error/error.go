package error

import "testing"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// AssertErrorMessage asserts that err.Error() is equal to wantMsg.
func AssertErrorMessage(t *testing.T, err error, wantMsg string) {
	if err == nil && wantMsg != "" {
		t.Errorf("did not get an error, but wanted error '%v'", wantMsg)
	}

	if err != nil && err.Error() != wantMsg {
		t.Errorf("got error '%v', but wanted error '%v'", err, wantMsg)
	}
}
