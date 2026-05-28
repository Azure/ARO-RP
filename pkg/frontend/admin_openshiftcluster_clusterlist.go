package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

type adminClusterListEntry struct {
	Key                     string `json:"key"`
	Name                    string `json:"name"`
	Subscription            string `json:"subscription"`
	ResourceGroup           string `json:"resourceGroup"`
	ResourceId              string `json:"resourceId"`
	ProvisioningState       string `json:"provisioningState"`
	FailedProvisioningState string `json:"failedProvisioningState"`
	Version                 string `json:"version"`
	CreatedAt               string `json:"createdAt"`
	CreatedBy               string `json:"createdBy"`
	ProvisionedBy           string `json:"provisionedBy"`
	LastModified            string `json:"lastModified"`
}

func (f *frontend) getAdminOpenShiftClusterList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	b, err := f._getAdminOpenShiftClusterList(ctx, r)
	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getAdminOpenShiftClusterList(ctx context.Context, r *http.Request) ([]byte, error) {
	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return nil, err
	}

	docs, err := dbOpenShiftClusters.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	var docList []*api.OpenShiftClusterDocument
	if docs != nil {
		docList = docs.OpenShiftClusterDocuments
	}

	filter := newAdminClusterListFilter(r)

	clusters := make([]*adminClusterListEntry, 0, len(docList))
	for _, doc := range docList {
		if doc.OpenShiftCluster == nil {
			continue
		}
		entry := newAdminClusterListEntry(doc)
		if filter != nil && !filter.matches(entry) {
			continue
		}
		clusters = append(clusters, entry)
	}

	sort.Slice(clusters, func(i, j int) bool { return clusters[i].ResourceId < clusters[j].ResourceId })

	return json.MarshalIndent(clusters, "", "    ")
}

type adminClusterListFilter struct {
	name          string
	subscription  string
	version       string
	createdBy     string
	provisionedBy string
	state         string
}

var adminClusterListFilterParams = []string{"name", "subscription", "version", "created_by", "provisioned_by", "state"}

func newAdminClusterListFilter(r *http.Request) *adminClusterListFilter {
	q := r.URL.Query()
	if !slices.ContainsFunc(adminClusterListFilterParams, q.Has) {
		return nil
	}

	return &adminClusterListFilter{
		name:          strings.ToLower(q.Get("name")),
		subscription:  strings.ToLower(q.Get("subscription")),
		version:       strings.ToLower(q.Get("version")),
		createdBy:     strings.ToLower(q.Get("created_by")),
		provisionedBy: strings.ToLower(q.Get("provisioned_by")),
		state:         strings.ToLower(q.Get("state")),
	}
}

func (af *adminClusterListFilter) matches(entry *adminClusterListEntry) bool {
	if af.name != "" && !strings.Contains(strings.ToLower(entry.Name), af.name) {
		return false
	}
	if af.subscription != "" && !strings.Contains(strings.ToLower(entry.Subscription), af.subscription) {
		return false
	}
	if af.version != "" && !strings.Contains(strings.ToLower(entry.Version), af.version) {
		return false
	}
	if af.createdBy != "" && !strings.Contains(strings.ToLower(entry.CreatedBy), af.createdBy) {
		return false
	}
	if af.provisionedBy != "" && !strings.Contains(strings.ToLower(entry.ProvisionedBy), af.provisionedBy) {
		return false
	}
	if af.state != "" && !strings.EqualFold(entry.ProvisioningState, af.state) {
		return false
	}
	return true
}

func newAdminClusterListEntry(doc *api.OpenShiftClusterDocument) *adminClusterListEntry {
	entry := &adminClusterListEntry{
		Key:                     doc.ID,
		ResourceId:              doc.OpenShiftCluster.ID,
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
	}

	if !doc.OpenShiftCluster.Properties.CreatedAt.IsZero() {
		entry.CreatedAt = doc.OpenShiftCluster.Properties.CreatedAt.Format(time.RFC3339)
	}

	if doc.OpenShiftCluster.SystemData.LastModifiedAt != nil {
		entry.LastModified = doc.OpenShiftCluster.SystemData.LastModifiedAt.Format(time.RFC3339)
	}

	return entry
}
