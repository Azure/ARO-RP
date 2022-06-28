package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

// /admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/skus
func (f *frontend) getAdminOpenShiftClusterVMResizeOptions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	r.URL.Path = filepath.Dir(r.URL.Path)

	b, err := f._getAdminOpenShiftClusterVMResizeOptions(ctx, r, log)

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getAdminOpenShiftClusterVMResizeOptions(ctx context.Context, r *http.Request, log *logrus.Entry) ([]byte, error) {
	vars := mux.Vars(r)

	resourceID := strings.TrimPrefix(r.URL.Path, "/admin")

	doc, err := f.dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
			"The Resource '%s/%s' under resource group '%s' was not found.",
			vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
	case err != nil:
		return nil, err
	}

	subscriptionDoc, err := f.getSubscriptionDocument(ctx, doc.Key)
	if err != nil {
		return nil, err
	}

	a, err := f.azureActionsFactory(log, f.env, doc.OpenShiftCluster, subscriptionDoc)
	if err != nil {
		return nil, err
	}

	skus, err := a.VMSizeList(ctx)
	if err != nil {
		return nil, err
	}

	return json.Marshal(f.filterVMSkus(skus))
}

func (f *frontend) filterVMSkus(skus []mgmtcompute.ResourceSku) []string {
	filteredSkus := []string{}

	for _, sku := range skus {
		if sku.Restrictions != nil && len(*sku.Restrictions) == 0 {
			if sku.Name != nil {
				filteredSkus = append(filteredSkus, *sku.Name)
			}
		}
	}

	return filteredSkus
}
