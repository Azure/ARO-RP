package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	HIVE_ENV_VARIABLE         = "HIVEKUBECONFIGPATH"
	HIVE_INSTALL_ENV_VARIABLE = "ARO_INSTALL_VIA_HIVE"
)

func (d *dev) HiveRestConfig(ctx context.Context, index int) (*rest.Config, error) {
	// Indexes above 0 have _index appended to them
	envVar := HIVE_ENV_VARIABLE
	if index != 0 {
		envVar = fmt.Sprintf("%s_%d", HIVE_ENV_VARIABLE, index)
	}

	kubeConfigPath := os.Getenv(envVar)
	if kubeConfigPath == "" {
		return nil, fmt.Errorf("missing %s env variable", HIVE_ENV_VARIABLE)
	}

	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}

	return restConfig, nil
}

func (d *dev) InstallViaHive(ctx context.Context) (bool, error) {
	installViaHive := os.Getenv(HIVE_INSTALL_ENV_VARIABLE)
	if installViaHive != "" {
		return true, nil
	}
	return false, nil
}
