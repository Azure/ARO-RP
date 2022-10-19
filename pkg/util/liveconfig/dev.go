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

func (d *dev) HiveRestConfig(ctx context.Context, index int, credentialType AksCredentialType) (*rest.Config, error) {
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

	// Hive shards are planned but not deployed or implemented yet
	d.hiveCredentialsMutex.RLock()
	credentialCache, exists := d.cachedCredentials[credentialType]
	if exists {
		cached, exists := credentialCache[index]
		d.hiveCredentialsMutex.RUnlock()
		if exists {
			return rest.CopyConfig(cached), nil
		}
	} else {
		d.hiveCredentialsMutex.RUnlock()
	}

	kubeConfig, err := getAksKubeconfig(ctx, d.managedClustersClient, d.location, index, credentialType)
	if err != nil {
		return nil, err
	}

	d.hiveCredentialsMutex.Lock()
	d.cachedCredentials[credentialType] = map[int]*rest.Config{
		index: kubeConfig,
	}
	d.hiveCredentialsMutex.Unlock()

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
