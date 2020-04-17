package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getAdminOpenShiftClusters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	b, err := f._getAdminOpenShiftClusters(ctx, r, f.apis[admin.APIVersion].OpenShiftClusterConverter())

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getAdminOpenShiftClusters(ctx context.Context, r *http.Request, converter api.OpenShiftClusterConverter) ([]byte, error) {
	var ocs []*api.OpenShiftCluster

	i := f.db.OpenShiftClusters.List()
	for {
		docs, err := i.Next(ctx, -1)
		if err != nil {
			return nil, err
		}
		if docs == nil {
			break
		}

		for _, doc := range docs.OpenShiftClusterDocuments {
			ocs = append(ocs, doc.OpenShiftCluster)
		}
	}

	for i := range ocs {
		ocs[i].Properties.ClusterProfile.PullSecret = ""
		ocs[i].Properties.ServicePrincipalProfile.ClientSecret = ""
	}

	// NOTE(ehashman): sort of a hack, must currently provide a nextPage link, so I left it blank
	return json.MarshalIndent(converter.ToExternalList(ocs, ""), "", "    ")
}
