package admin

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import "github.com/Azure/ARO-RP/pkg/api"

type openShiftClusterDocumentConverter struct{}

// ToExternalList returns a slice of external representations of the internal
// objects
func (c *openShiftClusterDocumentConverter) ToExternalList(docs []*api.OpenShiftClusterDocument, nextLink string) interface{} {
	l := &OpenShiftClusterList{
		OpenShiftClusters: make([]*OpenShiftCluster, 0, len(docs)),
		NextLink:          nextLink,
	}

	conv := &openShiftClusterConverter{}

	for _, doc := range docs {
		converted := conv.ToExternal(doc.OpenShiftCluster).(*OpenShiftCluster)
		converted.Properties.ClusterID = doc.ID
		l.OpenShiftClusters = append(l.OpenShiftClusters, converted)
	}

	return l
}

// ToExternal returns an external representation of the internal object
func (c *openShiftClusterDocumentConverter) ToExternal(doc *api.OpenShiftClusterDocument) interface{} {
	conv := &openShiftClusterConverter{}
	converted := conv.ToExternal(doc.OpenShiftCluster).(*OpenShiftCluster)
	converted.Properties.ClusterID = doc.ID
	return converted
}
