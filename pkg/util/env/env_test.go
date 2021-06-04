package env

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
)

type TestEnv struct {
	data map[string]string
}

func NewTestEnv(envMap map[string]string) TestEnv {
	return TestEnv{
		data: envMap,
	}
}

func (t *TestEnv) Getenv(key string) string {
	return t.data[key]
}
func (t *TestEnv) LookupEnv(key string) (string, bool) {
	if out, ok := t.data[key]; ok {
		return out, true
	} else {
		return "", false
	}
}

func TestLookupEnv(t *testing.T) {
	tests := []struct {
		name        string
		environment map[string]string
		lookup      string
		expect      bool
	}{
		{
			name:        "no environment",
			environment: make(map[string]string),
			lookup:      "keyThatDoesNotExist",
			expect:      false,
		},
		{
			name: "env but missing key",
			environment: map[string]string{
				"someKey": "someValue",
			},
			lookup: "keyThatDoesNotExist",
			expect: false,
		},
		{
			name: "lookup existing key",
			environment: map[string]string{
				"someKey": "someValue",
			},
			lookup: "someKey",
			expect: true,
		},
	}

	for _, test := range tests {
		testEnv := NewTestEnv(test.environment)
		_, got := testEnv.LookupEnv(test.lookup)
		if got != test.expect {
			t.Errorf("%s: expected %#v got %#v", test.name, test.expect, got)
		}
	}
}

func TestGetenv(t *testing.T) {
	tests := []struct {
		name        string
		environment map[string]string
		lookup      string
		expect      string
	}{
		{
			name:        "no environment",
			environment: make(map[string]string),
			lookup:      "keyThatDoesNotExist",
			expect:      "",
		},
		{
			name: "env but missing key",
			environment: map[string]string{
				"someKey": "someValue",
			},
			lookup: "keyThatDoesNotExist",
			expect: "",
		},
		{
			name: "lookup existing key",
			environment: map[string]string{
				"someKey": "someValue",
			},
			lookup: "someKey",
			expect: "someValue",
		},
	}

	for _, test := range tests {
		testEnv := NewTestEnv(test.environment)
		got := testEnv.Getenv(test.lookup)
		if got != test.expect {
			t.Errorf("%s: expected %#v got %#v", test.name, test.expect, got)
		}
	}
}
