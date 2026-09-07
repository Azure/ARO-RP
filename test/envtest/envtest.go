package envtest

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"os"
	"runtime"
	"testing"

	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const (
	USE_ENVTEST_VAR = "ARO_USE_ENVTEST"
	// See Makefile for the version that envtest will download
	ENVTEST_VERSION = "1.33.0"
)

func SkipIfNotUsingEnvtest(t *testing.T) {
	useEnvtest := os.Getenv(USE_ENVTEST_VAR)
	if useEnvtest != "true" {
		t.Skip(USE_ENVTEST_VAR, "is not set to 'true', skipping")
	}
}

// Start a testing environment using envtest and return the rest config for a
// Kubernetes client to use. The environment is automatically torn down at the
// end of the test.
func StartTestEnvironment(t *testing.T) *rest.Config {
	t.Helper()

	loc, err := envtest.SetupEnvtestDefaultBinaryAssetsDirectory()
	if err != nil {
		t.Fatal(err)
	}

	e := &envtest.Environment{
		BinaryAssetsDirectory: loc + "/" + ENVTEST_VERSION + "-" + runtime.GOOS + "-" + runtime.GOARCH,
	}

	t.Cleanup(func() {
		err := e.Stop()
		if err != nil {
			t.Fatal(err)
		}
	})
	restConfig, err := e.Start()
	if err != nil {
		t.Fatalf("error starting up envtest cluster: %s", err)
	}
	return restConfig
}
