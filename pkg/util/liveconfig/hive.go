package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	mgmtcontainerservice "github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2021-10-01/containerservice"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	hiveKubeconfigPathEnvVar  = "HIVE_KUBE_CONFIG_PATH"
	hiveInstallerEnableEnvVar = "ARO_INSTALL_VIA_HIVE"
	hiveDefaultPullSpecEnvVar = "ARO_HIVE_DEFAULT_INSTALLER_PULLSPEC"
	hiveAdoptEnableEnvVar     = "ARO_ADOPT_BY_HIVE"
)

func parseKubeconfig(credentials []mgmtcontainerservice.CredentialResult) (*rest.Config, error) {
	res := make([]byte, base64.StdEncoding.DecodedLen(len(*credentials[0].Value)))
	_, err := base64.StdEncoding.Decode(res, *credentials[0].Value)
	if err != nil {
		return nil, err
	}

	clientconfig, err := clientcmd.NewClientConfigFromBytes(res)
	if err != nil {
		return nil, err
	}

	restConfig, err := clientconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return restConfig, nil
}

func (p *prod) HiveRestConfig(ctx context.Context, index int) (*rest.Config, error) {
	// NOTE: This RWMutex locks on a fetch for any index for simplicity, rather
	// than a more granular per-index lock. As of the time of writing, multiple
	// Hive shards are planned but unimplemented elsewhere.
	p.hiveCredentialsMutex.RLock()
	cached, ext := p.cachedCredentials[index]
	p.hiveCredentialsMutex.RUnlock()
	if ext {
		return rest.CopyConfig(cached), nil
	}

	// Lock the RWMutex as we're starting to fetch so that new readers will wait
	// for the existing Azure API call to be done.
	p.hiveCredentialsMutex.Lock()
	defer p.hiveCredentialsMutex.Unlock()

	rpResourceGroup := fmt.Sprintf("rp-%s", p.location)
	rpResourceName := fmt.Sprintf("aro-aks-cluster-%03d", index)

	res, err := p.managedClustersClient.ListClusterUserCredentials(ctx, rpResourceGroup, rpResourceName, "")
	if err != nil {
		return nil, err
	}

	parsed, err := parseKubeconfig(*res.Kubeconfigs)
	if err != nil {
		return nil, err
	}

	p.cachedCredentials[index] = parsed
	return rest.CopyConfig(parsed), nil
}

func (p *prod) InstallViaHive(ctx context.Context) (bool, error) {
	// TODO: Replace with RP Live Service Config (KeyVault)
	installViaHive := os.Getenv(hiveInstallerEnableEnvVar)
	if installViaHive != "" {
		return true, nil
	}
	return false, nil
}

func (p *prod) DefaultInstallerPullSpecOverride(ctx context.Context) string {
	return os.Getenv(hiveDefaultPullSpecEnvVar)
}

func (p *prod) AdoptByHive(ctx context.Context) (bool, error) {
	// TODO: Replace with RP Live Service Config (KeyVault)
	adopt := os.Getenv(hiveAdoptEnableEnvVar)
	if adopt != "" {
		return true, nil
	}
	return false, nil
}
