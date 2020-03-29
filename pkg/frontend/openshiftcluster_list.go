package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getOpenShiftClusters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	b, err := f._getOpenShiftClusters(ctx, r, f.apis[vars["api-version"]].OpenShiftClusterConverter())

	reply(log, w, nil, b, err)
}

func (f *frontend) _getOpenShiftClusters(ctx context.Context, r *http.Request, converter api.OpenShiftClusterConverter) ([]byte, error) {
	vars := mux.Vars(r)

	prefix := "/subscriptions/" + vars["subscriptionId"] + "/"
	if vars["resourceGroupName"] != "" {
		prefix += "resourcegroups/" + vars["resourceGroupName"] + "/"
	}

	i, err := f.db.OpenShiftClusters.ListByPrefix(vars["subscriptionId"], prefix)
	if err != nil {
		return nil, err
	}

	var ocs []*api.OpenShiftCluster

	for {
		docs, err := i.Next(ctx)
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

	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	f.ocEnricher.Enrich(timeoutCtx, ocs...)

	for i := range ocs {
		ocs[i].Properties.ClusterProfile.PullSecret = ""
		ocs[i].Properties.ServicePrincipalProfile.ClientSecret = ""
	}

	return json.MarshalIndent(converter.ToExternalList(ocs), "", "    ")
}
