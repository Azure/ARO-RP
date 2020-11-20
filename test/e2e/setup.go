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
	projectv1client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	machineapiclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"

	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/insights"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/util/cluster"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/instancemetadata"
	"github.com/Azure/ARO-RP/test/util/kubeadminkubeconfig"
)

type clientSet struct {
	OpenshiftClusters redhatopenshift.OpenShiftClustersClient
	Operations        redhatopenshift.OperationsClient
	VirtualMachines   compute.VirtualMachinesClient
	Resources         features.ResourcesClient
	ActivityLogs      insights.ActivityLogsClient
	VirtualNetworks   network.VirtualNetworksClient

	RestConfig  *rest.Config
	Kubernetes  kubernetes.Interface
	MachineAPI  machineapiclient.Interface
	AROClusters aroclient.AroV1alpha1Interface
	Project     projectv1client.ProjectV1Interface
}

var (
	log            *logrus.Entry
	deploymentMode deployment.Mode
	im             instancemetadata.InstanceMetadata
	clusterName    string
	clients        *clientSet
)

func skipIfNotInDevelopmentEnv() {
	if deploymentMode != deployment.Development {
		Skip("skipping tests in non-development environment")
	}
}

func resourceIDFromEnv() string {
	return fmt.Sprintf(
		"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s",
		im.SubscriptionID(), im.ResourceGroup(), clusterName)
}

func newClientSet(ctx context.Context) (*clientSet, error) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	configv1, err := kubeadminkubeconfig.Get(ctx, log, im, authorizer, resourceIDFromEnv())
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

	projectcli, err := projectv1client.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	arocli, err := aroclient.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	return &clientSet{
		OpenshiftClusters: redhatopenshift.NewOpenShiftClustersClient(im.SubscriptionID(), authorizer),
		Operations:        redhatopenshift.NewOperationsClient(im.SubscriptionID(), authorizer),
		VirtualMachines:   compute.NewVirtualMachinesClient(im.SubscriptionID(), authorizer),
		Resources:         features.NewResourcesClient(im.SubscriptionID(), authorizer),
		ActivityLogs:      insights.NewActivityLogsClient(im.SubscriptionID(), authorizer),
		VirtualNetworks:   network.NewVirtualNetworksClient(im.SubscriptionID(), authorizer),

		RestConfig:  restconfig,
		Kubernetes:  cli,
		MachineAPI:  machineapicli,
		AROClusters: arocli,
		Project:     projectcli,
	}, nil
}

func setup(ctx context.Context) error {
	deploymentMode = deployment.NewMode()
	log.Infof("running in %s mode", deploymentMode)

	var err error
	im, err = instancemetadata.NewDev()
	if err != nil {
		return err
	}

	for _, key := range []string{
		"CLUSTER",
	} {
		if _, found := os.LookupEnv(key); !found {
			return fmt.Errorf("environment variable %q unset", key)
		}
	}

	clusterName = os.Getenv("CLUSTER")

	if os.Getenv("CI") != "" { // always create cluster in CI
		cluster, err := cluster.New(log, deploymentMode, im, os.Getenv("CI") != "")
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
		cluster, err := cluster.New(log, deploymentMode, im, os.Getenv("CI") != "")
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
