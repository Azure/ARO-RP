package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

type adminClusterListEntry struct {
	Key                     string `json:"key"`
	Name                    string `json:"name"`
	Subscription            string `json:"subscription"`
	ResourceGroup           string `json:"resourceGroup"`
	ResourceID              string `json:"resourceId"`
	ProvisioningState       string `json:"provisioningState"`
	FailedProvisioningState string `json:"failedProvisioningState"`
	Version                 string `json:"version"`
	CreatedAt               string `json:"createdAt"`
	CreatedBy               string `json:"createdBy"`
	ProvisionedBy           string `json:"provisionedBy"`
	LastModified            string `json:"lastModified"`
}

type adminClusterOverviewList struct {
	Clusters []*adminClusterListEntry `json:"value"`
	NextLink string                   `json:"nextLink,omitempty"`
}

func (f *frontend) getAdminOpenShiftClusters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		adminReply(log, w, nil, nil, err)
		return
	}

	if r.URL.Query().Get("view") == "overview" {
		b, err := f._getAdminOpenShiftClustersOverview(ctx, log, r, func(skipToken string) (cosmosdb.OpenShiftClusterDocumentIterator, error) {
			return dbOpenShiftClusters.List(skipToken), nil
		})
		adminReply(log, w, nil, b, err)
		return
	}

	b, err := f._getOpenShiftClusters(ctx, log, r, f.apis[admin.APIVersion].OpenShiftClusterConverter, func(skipToken string) (cosmosdb.OpenShiftClusterDocumentIterator, error) {
		return dbOpenShiftClusters.List(skipToken), nil
	})

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getAdminOpenShiftClustersOverview(ctx context.Context, log *logrus.Entry, r *http.Request, lister func(string) (cosmosdb.OpenShiftClusterDocumentIterator, error)) ([]byte, error) {
	skipToken, err := f.parseSkipToken(r.URL.String())
	if err != nil {
		return nil, err
	}

	i, err := lister(skipToken)
	if err != nil {
		return nil, err
	}

	docs, err := i.Next(ctx, 10)
	if err != nil {
		return nil, err
	}

	clusters := make([]*adminClusterListEntry, 0)
	if docs != nil {
		for _, doc := range docs.OpenShiftClusterDocuments {
			if doc.OpenShiftCluster == nil {
				continue
			}
			clusters = append(clusters, newAdminClusterListEntry(log, doc))
		}
	}

	nextLink, err := f.buildNextLink(r.Header.Get("Referer"), i.Continuation())
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(adminClusterOverviewList{
		Clusters: clusters,
		NextLink: nextLink,
	}, "", "    ")
}

func newAdminClusterListEntry(log *logrus.Entry, doc *api.OpenShiftClusterDocument) *adminClusterListEntry {
	entry := &adminClusterListEntry{
		Key:                     doc.ID,
		ResourceID:              doc.OpenShiftCluster.ID,
		Version:                 doc.OpenShiftCluster.Properties.ClusterProfile.Version,
		CreatedBy:               doc.OpenShiftCluster.Properties.CreatedBy,
		ProvisionedBy:           doc.OpenShiftCluster.Properties.ProvisionedBy,
		ProvisioningState:       doc.OpenShiftCluster.Properties.ProvisioningState.String(),
		FailedProvisioningState: doc.OpenShiftCluster.Properties.FailedProvisioningState.String(),
	}

	if resource, err := azure.ParseResourceID(doc.OpenShiftCluster.ID); err == nil {
		entry.Name = resource.ResourceName
		entry.Subscription = resource.SubscriptionID
		entry.ResourceGroup = resource.ResourceGroup
	} else {
		log.Warnf("failed to parse resource ID %q for document %q: %v", doc.OpenShiftCluster.ID, doc.ID, err)
	}

	if !doc.OpenShiftCluster.Properties.CreatedAt.IsZero() {
		entry.CreatedAt = doc.OpenShiftCluster.Properties.CreatedAt.Format(time.RFC3339)
	}

	if doc.OpenShiftCluster.SystemData.LastModifiedAt != nil {
		entry.LastModified = doc.OpenShiftCluster.SystemData.LastModifiedAt.Format(time.RFC3339)
	}

	return entry
}
