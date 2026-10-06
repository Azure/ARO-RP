package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
)

type AdminOpenShiftCluster struct {
	Key                     string `json:"key"`
	Name                    string `json:"name"`
	Subscription            string `json:"subscription"`
	ResourceGroup           string `json:"resourceGroup"`
	ResourceId              string `json:"resourceId"`
	ProvisioningState       string `json:"provisioningState"`
	FailedProvisioningState string `json:"failedprovisioningState"`
	Version                 string `json:"version"`
	CreatedAt               string `json:"createdAt"`
	LastModified            string `json:"lastModified"`
	ProvisionedBy           string `json:"provisionedBy"`
}

func (p *portal) clusters(w http.ResponseWriter, r *http.Request) {
	dbOpenShiftClusters, err := p.dbGroup.OpenShiftClusters()
	if err != nil {
		p.internalServerError(w, err)
		return
	}
	ctx := r.Context()

	docs, err := dbOpenShiftClusters.ListAll(ctx)
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	clusters := make([]*AdminOpenShiftCluster, 0, len(docs.OpenShiftClusterDocuments))
	for _, doc := range docs.OpenShiftClusterDocuments {
		if doc.OpenShiftCluster == nil {
			continue
		}

		ps := doc.OpenShiftCluster.Properties.ProvisioningState
		fps := doc.OpenShiftCluster.Properties.FailedProvisioningState
		subscription := "Unknown"
		resourceGroup := "Unknown"
		name := "Unknown"

		if resource, err := azure.ParseResourceID(doc.OpenShiftCluster.ID); err == nil {
			subscription = resource.SubscriptionID
			resourceGroup = resource.ResourceGroup
			name = resource.ResourceName
		}

		createdAt := "Unknown"
		if !doc.OpenShiftCluster.Properties.CreatedAt.IsZero() {
			createdAt = doc.OpenShiftCluster.Properties.CreatedAt.Format(time.RFC3339)
		}

		lastModified := "Unknown"
		if doc.OpenShiftCluster.SystemData.LastModifiedAt != nil {
			lastModified = doc.OpenShiftCluster.SystemData.LastModifiedAt.Format(time.RFC3339)
		}

		clusters = append(clusters, &AdminOpenShiftCluster{
			Key:                     doc.ID,
			ResourceId:              doc.OpenShiftCluster.ID,
			Name:                    name,
			Subscription:            subscription,
			ResourceGroup:           resourceGroup,
			Version:                 doc.OpenShiftCluster.Properties.ClusterProfile.Version,
			CreatedAt:               createdAt,
			LastModified:            lastModified,
			ProvisionedBy:           doc.OpenShiftCluster.Properties.ProvisionedBy,
			ProvisioningState:       ps.String(),
			FailedProvisioningState: fps.String(),
		})
	}

	sort.SliceStable(clusters, func(i, j int) bool { return strings.Compare(clusters[i].Key, clusters[j].Key) < 0 })

	b, err := json.MarshalIndent(clusters, "", "    ")
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}
