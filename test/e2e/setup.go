package e2e

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"log"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/zhuoli/ARO-RP/test/e2e/testresources"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/redhatopenshift"
)

type ClientSet struct {
	OpenshiftClusters redhatopenshift.OpenShiftClustersClient
	Operations        redhatopenshift.OperationsClient
	Kubernetes        kubernetes.Interface
}

var (
	Log     *logrus.Entry
	Clients *ClientSet
)

func newClientSet() (*ClientSet, error) {
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

	cli, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	return &ClientSet{
		OpenshiftClusters: redhatopenshift.NewOpenShiftClustersClient(os.Getenv("AZURE_SUBSCRIPTION_ID"), authorizer),
		Operations:        redhatopenshift.NewOperationsClient(os.Getenv("AZURE_SUBSCRIPTION_ID"), authorizer),
		Kubernetes:        cli,
	}, nil
}

var _ = BeforeSuite(func() {
	Log.Info("BeforeSuite")
	for _, key := range []string{
		"AZURE_SUBSCRIPTION_ID",
		"CLUSTER",
		"RESOURCEGROUP",
	} {
		if _, found := os.LookupEnv(key); !found {
			panic(fmt.Sprintf("environment variable %q unset", key))
		}
	}

	var err error
	Clients, err = newClientSet()
	if err != nil {
		panic(err)
	}

	By("add cluster blob for billing e2es", func() {
		subID := os.Getenv("AZURE_SUBSCRIPTION_ID")
		resourceGroup := os.Getenv("RESOURCEGROUP")
		resourceName := os.Getenv("CLUSTER")

		log.Printf("Calling AddClusterBlobForBillingE2E with subID : %s, resourceGroup : %s, resourceName : %s", subID, resourceGroup, resourceName)
		err := testresources.AddClusterBlobForBillingE2E(subID, resourceGroup, resourceName)
		Expect(err).To(BeNil())
	})

})
