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
	envVar := hiveKubeconfigPathEnvVar
	if index != 0 {
		envVar = fmt.Sprintf("%s_%d", hiveKubeconfigPathEnvVar, index)
	}

	// Use an override kubeconfig path if one is provided
	kubeConfigPath := os.Getenv(envVar)
	if kubeConfigPath != "" {
		restConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			return nil, err
		}
		return restConfig, nil
	}

	// NOTE: This RWMutex locks on a fetch for any index for simplicity, rather
	// than a more granular per-index lock. As of the time of writing, multiple
	// Hive shards are planned but unimplemented elsewhere.
	d.hiveCredentialsMutex.RLock()
	cached, ext := d.cachedCredentials[index]
	d.hiveCredentialsMutex.RUnlock()
	if ext {
		return rest.CopyConfig(cached), nil
	}

	// Lock the RWMutex as we're starting to fetch so that new readers will wait
	// for the existing Azure API call to be done.
	d.hiveCredentialsMutex.Lock()
	defer d.hiveCredentialsMutex.Unlock()

	kubeConfig, err := getAksKubeconfig(ctx, d.managedClustersClient, index, d.location)
	if err != nil {
		return nil, err
	}

	d.cachedCredentials[index] = kubeConfig
	return rest.CopyConfig(kubeConfig), nil
}

func (d *dev) InstallViaHive(ctx context.Context) (bool, error) {
	installViaHive := os.Getenv(hiveInstallerEnableEnvVar)
	if installViaHive != "" {
		return true, nil
	}
	return false, nil
}

func (d *dev) DefaultInstallerPullSpecOverride(ctx context.Context) string {
	return os.Getenv(hiveDefaultPullSpecEnvVar)
}

func (p *dev) AdoptByHive(ctx context.Context) (bool, error) {
	adopt := os.Getenv(hiveAdoptEnableEnvVar)
	if adopt != "" {
		return true, nil
	}
	return false, nil
}
