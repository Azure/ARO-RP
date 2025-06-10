package liveconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"strings"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	armcontainerservice "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6"

	utilcontainerservice "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armcontainerservice"
)

func getAksClusterByNameAndLocation(ctx context.Context, aksClusters *runtime.Pager[armcontainerservice.ManagedClustersClientListResponse], aksClusterName, location string) (*armcontainerservice.ManagedCluster, error) {
	for aksClusters.More() {
		nr, err := aksClusters.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, cluster := range nr.Value {
			if strings.EqualFold(*cluster.Name, aksClusterName) && strings.EqualFold(*cluster.Location, location) {
				return cluster, nil
			}
		}
	}
	return nil, nil
}

func getAksShardKubeconfig(ctx context.Context, managedClustersClient utilcontainerservice.ManagedClustersClient, location string, shard int) (*rest.Config, error) {
	aksClusterName := fmt.Sprintf("aro-aks-cluster-%03d", shard)

	aksClusters := managedClustersClient.List(ctx)

	aksCluster, err := getAksClusterByNameAndLocation(ctx, aksClusters, aksClusterName, location)
	if err != nil {
		return nil, err
	}
	if aksCluster == nil {
		return nil, fmt.Errorf("failed to find the AKS cluster %s in %s", aksClusterName, location)
	}

	aksResourceGroup := strings.Replace(*aksCluster.Properties.NodeResourceGroup, fmt.Sprintf("-aks%d", shard), "", 1)

	res, err := managedClustersClient.ListClusterAdminCredentials(ctx, aksResourceGroup, aksClusterName, "public")
	if err != nil {
		return nil, err
	}

	return parseKubeconfig(res.Kubeconfigs)
}

func parseKubeconfig(credentials []*armcontainerservice.CredentialResult) (*rest.Config, error) {
	clientconfig, err := clientcmd.NewClientConfigFromBytes(credentials[0].Value)
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

func (p *prod) InstallViaHive(ctx context.Context) (bool, error) {
	// TODO: Replace with RP Live Service Config (KeyVault)
	installViaHive := os.Getenv(hiveInstallerEnableEnvVar)
	if installViaHive != "" {
		return true, nil
	}
	return false, nil
}

func (p *prod) DefaultInstallerPullSpecOverride(ctx context.Context) string {
	// TODO: we should probably not have an override in prod, but it may have unintended
	// consequences in an int-like development RP
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
