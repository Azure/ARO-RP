package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/feature"
)

func determineOutboundType(ctx context.Context, doc *api.OpenShiftClusterDocument, subscription *api.SubscriptionDocument) {
	// Honor the value of OutboundType if it was passed in via the API
	if doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType == "" {
		// Determine if this is a cluster with user defined routing
		doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType = api.OutboundTypeLoadbalancer
		if doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPrivate &&
			doc.OpenShiftCluster.Properties.IngressProfiles[0].Visibility == api.VisibilityPrivate &&
			feature.IsRegisteredForFeature(subscription.Subscription.Properties, api.FeatureFlagUserDefinedRouting) {
			doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType = api.OutboundTypeUserDefinedRouting
		}
	}
}
