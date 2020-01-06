package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getOpenShiftCluster(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(middleware.ContextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	b, err := f._getOpenShiftCluster(r, f.apis[vars["api-version"]].OpenShiftClusterConverter())

	reply(log, w, nil, b, err)
}

func (f *frontend) _getOpenShiftCluster(r *http.Request, converter api.OpenShiftClusterConverter) ([]byte, error) {
	vars := mux.Vars(r)

	doc, err := f.db.OpenShiftClusters.Get(r.URL.Path)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
	case err != nil:
		return nil, err
	}

	doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret = ""

	return json.MarshalIndent(converter.ToExternal(doc.OpenShiftCluster), "", "    ")
}
