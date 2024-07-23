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

func (d *dev) HiveRestConfig(ctx context.Context, shard int) (*rest.Config, error) {
	// shards above 0 have _shard appended to it
	envVar := hiveKubeconfigPathEnvVar
	if shard != 0 {
		envVar = fmt.Sprintf("%s_%d", hiveKubeconfigPathEnvVar, shard)
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

	// Hive shards are planned but not deployed or implemented yet
	d.hiveCredentialsMutex.RLock()
	cached, exists := d.cachedCredentials[shard]
	d.hiveCredentialsMutex.RUnlock()
	if exists {
		return rest.CopyConfig(cached), nil
	}

	// Lock the RWMutex as we're starting to fetch so that new readers will wait
	// for the existing Azure API call to be done.
	d.hiveCredentialsMutex.Lock()
	defer d.hiveCredentialsMutex.Unlock()

	kubeConfig, err := getAksShardKubeconfig(ctx, d.managedClustersClient, d.location, shard)
	if err != nil {
		return nil, err
	}

	d.cachedCredentials[shard] = kubeConfig

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
