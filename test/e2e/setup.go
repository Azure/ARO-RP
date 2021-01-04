package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	projectclient "github.com/openshift/client-go/project/clientset/versioned"
	machineapiclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"

	"github.com/Azure/ARO-RP/pkg/env"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/insights"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	openshiftclustersv20200430 "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2020-04-30/redhatopenshift"
	openshiftclustersv20210131preview "github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift/2021-01-31-preview/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/util/cluster"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/test/util/kubeadminkubeconfig"
)

type clientSet struct {
	OpenshiftClustersv20200430        openshiftclustersv20200430.OpenShiftClustersClient
	Operationsv20200430               openshiftclustersv20200430.OperationsClient
	OpenshiftClustersv20210131preview openshiftclustersv20210131preview.OpenShiftClustersClient
	Operationsv20210131preview        openshiftclustersv20210131preview.OperationsClient

	VirtualMachines compute.VirtualMachinesClient
	Resources       features.ResourcesClient
	ActivityLogs    insights.ActivityLogsClient
	VirtualNetworks network.VirtualNetworksClient

	RestConfig  *rest.Config
	Kubernetes  kubernetes.Interface
	MachineAPI  machineapiclient.Interface
	AROClusters aroclient.Interface
	Project     projectclient.Interface
}

var (
	log         *logrus.Entry
	_env        env.Core
	clusterName string
	clients     *clientSet
)

func skipIfNotInDevelopmentEnv() {
	if _env.DeploymentMode() != deployment.Development {
		Skip("skipping tests in non-development environment")
	}
}

func resourceIDFromEnv() string {
	return fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s",
		_env.SubscriptionID(), _env.ResourceGroup(), clusterName)
}

func newClientSet(ctx context.Context) (*clientSet, error) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	configv1, err := kubeadminkubeconfig.Get(ctx, log, _env, authorizer, resourceIDFromEnv())
	if err != nil {
		return nil, err
	}

	var config api.Config
	err = latest.Scheme.Convert(configv1, &config, nil)
	if err != nil {
		return nil, err
	}

	kubeconfig := clientcmd.NewDefaultClientConfig(config, &clientcmd.ConfigOverrides{})

	restconfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	cli, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	machineapicli, err := machineapiclient.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	projectcli, err := projectclient.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	arocli, err := aroclient.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	return &clientSet{
		OpenshiftClustersv20200430:        openshiftclustersv20200430.NewOpenShiftClustersClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Operationsv20200430:               openshiftclustersv20200430.NewOperationsClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		OpenshiftClustersv20210131preview: openshiftclustersv20210131preview.NewOpenShiftClustersClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Operationsv20210131preview:        openshiftclustersv20210131preview.NewOperationsClient(_env.Environment(), _env.SubscriptionID(), authorizer),

		VirtualMachines: compute.NewVirtualMachinesClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		Resources:       features.NewResourcesClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		ActivityLogs:    insights.NewActivityLogsClient(_env.Environment(), _env.SubscriptionID(), authorizer),
		VirtualNetworks: network.NewVirtualNetworksClient(_env.Environment(), _env.SubscriptionID(), authorizer),

		RestConfig:  restconfig,
		Kubernetes:  cli,
		MachineAPI:  machineapicli,
		AROClusters: arocli,
		Project:     projectcli,
	}, nil
}

func setup(ctx context.Context) error {
	for _, key := range []string{
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_TENANT_ID",
		"CLUSTER",
		"LOCATION",
		"RESOURCEGROUP",
	} {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	var err error
	_env, err = env.NewCoreForCI(ctx, log)
	if err != nil {
		return err
	}

	clusterName = os.Getenv("CLUSTER")

	if os.Getenv("CI") != "" { // always create cluster in CI
		cluster, err := cluster.New(log, _env, os.Getenv("CI") != "")
		if err != nil {
			return err
		}

		err = cluster.Create(ctx, clusterName)
		if err != nil {
			return err
		}
	}

	clients, err = newClientSet(ctx)
	if err != nil {
		return err
	}

	return nil
}

func done(ctx context.Context) error {
	// terminate early if delete flag is set to false
	if os.Getenv("CI") != "" && os.Getenv("E2E_DELETE_CLUSTER") != "false" {
		cluster, err := cluster.New(log, _env, os.Getenv("CI") != "")
		if err != nil {
			return err
		}

		err = cluster.Delete(ctx, clusterName)
		if err != nil {
			return err
		}
	}

	return nil
}

var _ = BeforeSuite(func() {
	log.Info("BeforeSuite")

	SetDefaultEventuallyTimeout(5 * time.Minute)
	SetDefaultEventuallyPollingInterval(10 * time.Second)

	if err := setup(context.Background()); err != nil {
		panic(err)
	}
})

var _ = AfterSuite(func() {
	log.Info("AfterSuite")

	if err := done(context.Background()); err != nil {
		panic(err)
	}
})
