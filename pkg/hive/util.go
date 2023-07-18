package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"

	"github.com/Azure/ARO-RP/pkg/api"
)

// See https://github.com/openshift/hive/blob/master/docs/using-hive.md#azure
// and github.com/openshift/installer/pkg/asset/installconfig/azure
func clusterSPToBytes(subscriptionDoc *api.SubscriptionDocument, oc *api.OpenShiftCluster) ([]byte, error) {
	return json.Marshal(map[string]string{
		"tenantId":       subscriptionDoc.Subscription.Properties.TenantID,
		"subscriptionId": subscriptionDoc.ID,
		"clientId":       oc.Properties.ServicePrincipalProfile.ClientID,
		"clientSecret":   string(oc.Properties.ServicePrincipalProfile.ClientSecret),
	})
}
