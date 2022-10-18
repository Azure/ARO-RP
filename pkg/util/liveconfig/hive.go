package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"strings"

	mgmtcontainerservice "github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2021-10-01/containerservice"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/containerservice"
)

const (
	hiveKubeconfigPathEnvVar  = "HIVE_KUBE_CONFIG_PATH"
	hiveInstallerEnableEnvVar = "ARO_INSTALL_VIA_HIVE"
	hiveDefaultPullSpecEnvVar = "ARO_HIVE_DEFAULT_INSTALLER_PULLSPEC"
	hiveAdoptEnableEnvVar     = "ARO_ADOPT_BY_HIVE"
)

func getAksKubeconfig(ctx context.Context, managedClustersClient containerservice.ManagedClustersClient, index int, location string) (*rest.Config, error) {
	aksClusterName := fmt.Sprintf("aro-aks-cluster-%03d", index)

	aksClusters, err := managedClustersClient.List(ctx)
	if err != nil {
		return nil, err
	}

	var aksCluster *mgmtcontainerservice.ManagedCluster

outerLoop:
	for aksClusters.NotDone() {
		for _, cluster := range aksClusters.Values() {
			if *cluster.Name == aksClusterName && *cluster.Location == location {
				aksCluster = &cluster
				break outerLoop
			}
		}
		err = aksClusters.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	if aksCluster == nil {
		return nil, fmt.Errorf("failed to find the AKS cluster %s in %s", aksClusterName, location)
	}

	aksResourceGroup := strings.Replace(*aksCluster.NodeResourceGroup, fmt.Sprintf("-aks%d", index), "", 1)

	res, err := managedClustersClient.ListClusterAdminCredentials(ctx, aksResourceGroup, aksClusterName, "public")
	if err != nil {
		return nil, err
	}

	return parseKubeconfig(*res.Kubeconfigs)
}

func parseKubeconfig(credentials []mgmtcontainerservice.CredentialResult) (*rest.Config, error) {
	clientconfig, err := clientcmd.NewClientConfigFromBytes(*credentials[0].Value)
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

	kubeConfig, err := getAksKubeconfig(ctx, p.managedClustersClient, index, p.location)
	if err != nil {
		return nil, err
	}

	p.cachedCredentials[index] = kubeConfig
	return rest.CopyConfig(kubeConfig), nil
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
