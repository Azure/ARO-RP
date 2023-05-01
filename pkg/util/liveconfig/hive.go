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

func getAksClusterByNameAndLocation(ctx context.Context, aksClusters mgmtcontainerservice.ManagedClusterListResultPage, aksClusterName, location string) (*mgmtcontainerservice.ManagedCluster, error) {
	for aksClusters.NotDone() {
		for _, cluster := range aksClusters.Values() {
			if strings.EqualFold(*cluster.Name, aksClusterName) && strings.EqualFold(*cluster.Location, location) {
				return &cluster, nil
			}
		}
		err := aksClusters.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func getAksShardKubeconfig(ctx context.Context, managedClustersClient containerservice.ManagedClustersClient, location string, shard int) (*rest.Config, error) {
	aksClusterName := fmt.Sprintf("aro-aks-cluster-%03d", shard)

	aksClusters, err := managedClustersClient.List(ctx)
	if err != nil {
		return nil, err
	}

	aksCluster, err := getAksClusterByNameAndLocation(ctx, aksClusters, aksClusterName, location)
	if err != nil {
		return nil, err
	}
	if aksCluster == nil {
		return nil, fmt.Errorf("failed to find the AKS cluster %s in %s", aksClusterName, location)
	}

	aksResourceGroup := strings.Replace(*aksCluster.NodeResourceGroup, fmt.Sprintf("-aks%d", shard), "", 1)

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

func (p *prod) HiveRestConfig(ctx context.Context, shard int) (*rest.Config, error) {
	// Hive shards are planned but not implemented yet
	p.hiveCredentialsMutex.RLock()
	cached, exists := p.cachedCredentials[shard]
	p.hiveCredentialsMutex.RUnlock()
	if exists {
		return rest.CopyConfig(cached), nil
	}

	// Lock the RWMutex as we're starting to fetch so that new readers will wait
	// for the existing Azure API call to be done.
	p.hiveCredentialsMutex.Lock()
	defer p.hiveCredentialsMutex.Unlock()

	kubeConfig, err := getAksShardKubeconfig(ctx, p.managedClustersClient, p.location, shard)
	if err != nil {
		return nil, err
	}

	p.cachedCredentials[shard] = kubeConfig

	return rest.CopyConfig(kubeConfig), nil
}

func (p *prod) InstallStrategy(ctx context.Context) (InstallStrategy, error) {
	// use the old variable first for compat
	installViaHive := os.Getenv(hiveInstallerEnableEnvVar)
	if installViaHive != "" {
		return HiveStrategy, nil
	}

	installStrategy := strings.ToLower(os.Getenv(installStrategyEnvVar))
	switch installStrategy {
	case "hive":
		return HiveStrategy, nil
	case "":
	case "builtin":
		return BuiltinStrategy, nil
	case "aks":
		return AKSStrategy, nil
	}
	return 0, fmt.Errorf("%s is not an install strategy", installStrategy)
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
