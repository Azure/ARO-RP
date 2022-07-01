package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const HIVEENVVARIABLE = "HIVEKUBECONFIGPATH"

func HiveRestConfig() (*rest.Config, error) {
	//only one for now
	kubeConfigPath := os.Getenv(HIVEENVVARIABLE)
	if kubeConfigPath == "" {
		return nil, fmt.Errorf("missing %s env variable", HIVEENVVARIABLE)
	}

	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}

	return restConfig, nil
}
