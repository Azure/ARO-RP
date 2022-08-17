package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"os"

	icazure "github.com/openshift/installer/pkg/asset/installconfig/azure"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Azure/ARO-RP/pkg/api"
)

const hiveKubeConfigEnvVar = "HIVEKUBECONFIGPATH"

func HiveRestConfig() (*rest.Config, error) {
	kubeConfigPath := os.Getenv(hiveKubeConfigEnvVar)
	if kubeConfigPath == "" {
		return nil, fmt.Errorf("missing %s env variable", hiveKubeConfigEnvVar)
	}

	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}

	return restConfig, nil
}

func clusterSPToBytes(subscriptionDoc *api.SubscriptionDocument, oc *api.OpenShiftCluster) ([]byte, error) {
	return json.Marshal(icazure.Credentials{
		TenantID:       subscriptionDoc.Subscription.Properties.TenantID,
		SubscriptionID: subscriptionDoc.ID,
		ClientID:       oc.Properties.ServicePrincipalProfile.ClientID,
		ClientSecret:   string(oc.Properties.ServicePrincipalProfile.ClientSecret),
	})
}
