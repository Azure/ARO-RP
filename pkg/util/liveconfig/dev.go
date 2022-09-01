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

func (d *dev) HiveRestConfig(ctx context.Context, index int) (*rest.Config, error) {
	// Indexes above 0 have _index appended to them
	envVar := HIVE_KUBE_CONFIG_PATH
	if index != 0 {
		envVar = fmt.Sprintf("%s_%d", HIVE_KUBE_CONFIG_PATH, index)
	}

	kubeConfigPath := os.Getenv(envVar)
	if kubeConfigPath == "" {
		return nil, fmt.Errorf("missing %s env variable", HIVE_KUBE_CONFIG_PATH)
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

func (d *dev) DefaultInstallerPullSpecOverride(ctx context.Context) string {
	return os.Getenv(HIVE_DEFAULT_INSTALLER_VARIABLE)
}
