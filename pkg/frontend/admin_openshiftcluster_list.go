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
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getAdminOpenShiftClusters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	b, err := f._getAdminOpenShiftClusters(ctx, r, f.apis["admin"].OpenShiftClusterConverter())

	reply(log, w, nil, b, err)
}

func (f *frontend) _getAdminOpenShiftClusters(ctx context.Context, r *http.Request, converter api.OpenShiftClusterConverter) ([]byte, error) {
	i, err := f.db.OpenShiftClusters.List()
	if err != nil {
		return nil, err
	}

	docs, err := i.Next(ctx, 10)
	if err != nil {
		return nil, err
	}

	var ocs []*api.OpenShiftCluster
	if docs != nil {
		for _, doc := range docs.OpenShiftClusterDocuments {
			ocs = append(ocs, doc.OpenShiftCluster)
		}
	}

	for i := range ocs {
		ocs[i].Properties.ClusterProfile.PullSecret = ""
		ocs[i].Properties.ServicePrincipalProfile.ClientSecret = ""
	}

	nextLink, err := f.buildNextLink(r.Header.Get("Referer"), i.Continuation())
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(converter.ToExternalList(ocs, nextLink), "", "    ")
}
