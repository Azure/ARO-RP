package json

import (
	"encoding/json"
	"testing"

	"github.com/Azure/ARO-RP/pkg/util/cmp"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

// Compare expected and actual JSON responses via unmarshaling into a map,
// to avoid issues with e.g. field ordering
func AssertJsonMatches(t *testing.T, want, got []byte) {
	t.Helper()
	var wantMap, gotMap any
	if err := json.Unmarshal(want, &wantMap); err != nil {
		t.Error(err)
	}
	if err := json.Unmarshal(got, &gotMap); err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(wantMap, gotMap); diff != "" {
		t.Error(diff)
	}
}
