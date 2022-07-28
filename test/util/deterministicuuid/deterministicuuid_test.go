package deterministicuuid

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
)

func TestDeterministicUUID(t *testing.T) {
	g := &gen{}

	// generate until it's obvious it's base-16 :)
	uuids := []string{
		"00000000-0000-0000-0000-000000000001",
		"00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003",
		"00000000-0000-0000-0000-000000000004",
		"00000000-0000-0000-0000-000000000005",
		"00000000-0000-0000-0000-000000000006",
		"00000000-0000-0000-0000-000000000007",
		"00000000-0000-0000-0000-000000000008",
		"00000000-0000-0000-0000-000000000009",
		"00000000-0000-0000-0000-00000000000a",
		"00000000-0000-0000-0000-00000000000b",
		"00000000-0000-0000-0000-00000000000c",
		"00000000-0000-0000-0000-00000000000d",
		"00000000-0000-0000-0000-00000000000e",
		"00000000-0000-0000-0000-00000000000f",
		"00000000-0000-0000-0000-000000000010",
	}

	for _, u := range uuids {
		genned := g.Generate()
		if genned != u {
			t.Error(u, genned)
		}
	}

	g.counter = 256
	genned := g.Generate()
	if genned != "00000000-0000-0000-0000-000000000101" {
		t.Errorf("not bitshifted correctly: %s", genned)
	}

	namespaced := &gen{namespace: 12}
	genned = namespaced.Generate()
	if genned != "0c0c0c0c-0c0c-0c0c-0c0c-0c0c0c0c0001" {
		t.Errorf("not namespaced correctly: %s", genned)
	}
}
