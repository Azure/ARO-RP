package main

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"testing"
)

func TestEnsureAroTag(t *testing.T) {
	/*
		This file should always fail this test when "+build !aro".
		The 'aro' tag is required for the openshift/installer to disable certain
		functionality which are valid for OpenShift on Azure, but not valid for ARO deployments.
		Related: https://github.com/openshift/installer/pull/4843
	*/
	// TODO: Use `azuretypes.Platform.IsARO()` from github.com/openshift/installer/pkg/types/azure
	if !platformIsAro {
		t.Fatalf("ARO-RP must be built, run, and tested with '-tags aro' to support github.com/openshift/installer, see https://github.com/openshift/installer/pull/4843/files")
	}
}
