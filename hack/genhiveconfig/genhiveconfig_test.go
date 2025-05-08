package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/Azure/ARO-RP/pkg/hive/failure"
)

func TestFailureReasonToInstallLogRegex(t *testing.T) {
	input := failure.InstallFailingReason{
		Name:    "TestReason",
		Reason:  "AzureTestReason",
		Message: "This is a sentence.",
		SearchRegexes: []*regexp.Regexp{
			regexp.MustCompile(".*"),
			regexp.MustCompile("^$"),
		},
	}

	want := installLogRegex{
		Name:                  "TestReason",
		InstallFailingReason:  "AzureTestReason",
		InstallFailingMessage: "This is a sentence.",
		SearchRegexStrings:    []string{".*", "^$"},
	}

	got := failureReasonToInstallLogRegex(input)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
