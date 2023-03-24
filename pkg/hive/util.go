package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"

	icazure "github.com/openshift/installer/pkg/asset/installconfig/azure"

	"github.com/Azure/ARO-RP/pkg/api"
)

func clusterSPToBytes(subscriptionDoc *api.SubscriptionDocument, oc *api.OpenShiftCluster) ([]byte, error) {
	return json.Marshal(icazure.Credentials{
		TenantID:       subscriptionDoc.Subscription.Properties.TenantID,
		SubscriptionID: subscriptionDoc.ID,
		ClientID:       oc.Properties.ServicePrincipalProfile.ClientID,
		ClientSecret:   string(oc.Properties.ServicePrincipalProfile.ClientSecret),
	})
}
