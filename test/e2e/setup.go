package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
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
	"k8s.io/client-go/tools/clientcmd"

	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/insights"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
)

type clientSet struct {
	OpenshiftClusters redhatopenshift.OpenShiftClustersClient
	Operations        redhatopenshift.OperationsClient
	VirtualMachines   compute.VirtualMachinesClient
	Resources         features.ResourcesClient
	ActivityLogs      insights.ActivityLogsClient

	Kubernetes  kubernetes.Interface
	MachineAPI  machineapiclient.Interface
	AROClusters aroclient.AroV1alpha1Interface
	Project     projectv1client.ProjectV1Interface
}

var (
	log     *logrus.Entry
	clients *clientSet
)

func skipIfNotInDevelopmentEnv() {
	if deployment.NewMode() != deployment.Development {
		Skip("skipping tests in non-development environment")
	}
}

func resourceIDFromEnv() string {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	resourceGroup := os.Getenv("RESOURCEGROUP")
	clusterName := os.Getenv("CLUSTER")
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s", subscriptionID, resourceGroup, clusterName)
}

func newClientSet(subscriptionID string) (*clientSet, error) {
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restconfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	cli := kubernetes.NewForConfigOrDie(restconfig)
	machineapicli := machineapiclient.NewForConfigOrDie(restconfig)

	arocli, err := aroclient.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	return &clientSet{
		OpenshiftClusters: redhatopenshift.NewOpenShiftClustersClient(subscriptionID, authorizer),
		Operations:        redhatopenshift.NewOperationsClient(subscriptionID, authorizer),
		VirtualMachines:   compute.NewVirtualMachinesClient(subscriptionID, authorizer),
		Resources:         features.NewResourcesClient(subscriptionID, authorizer),
		ActivityLogs:      insights.NewActivityLogsClient(subscriptionID, authorizer),

		Kubernetes:  cli,
		MachineAPI:  machineapicli,
		AROClusters: arocli,
		Project:     projectv1client.NewForConfigOrDie(restconfig),
	}, nil
}

var _ = BeforeSuite(func() {
	log.Info("BeforeSuite")

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(10 * time.Second)

	for _, key := range []string{
		"AZURE_SUBSCRIPTION_ID",
		"CLUSTER",
		"RESOURCEGROUP",
	} {
		if _, found := os.LookupEnv(key); !found {
			panic(fmt.Sprintf("environment variable %q unset", key))
		}
	}

	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")

	var err error
	clients, err = newClientSet(subscriptionID)
	if err != nil {
		panic(err)
	}
})
