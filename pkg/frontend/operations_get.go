package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getOperations(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(middleware.ContextKeyLog).(*logrus.Entry)

	l := &api.OperationList{
		Operations: []api.Operation{
			{
				Name: "Microsoft.RedHatOpenShift/locations/operationresults/read",
				Display: api.Display{
					Provider:  "Azure Red Hat OpenShift",
					Resource:  "locations/operationresults",
					Operation: "Read operation results",
				},
				Origin: "user,system",
			},
			{
				Name: "Microsoft.RedHatOpenShift/locations/operationsstatus/read",
				Display: api.Display{
					Provider:  "Azure Red Hat OpenShift",
					Resource:  "locations/operationsstatus",
					Operation: "Read operations status",
				},
				Origin: "user,system",
			},
			{
				Name: "Microsoft.RedHatOpenShift/openShiftClusters/read",
				Display: api.Display{
					Provider:  "Azure Red Hat OpenShift",
					Resource:  "openShiftClusters",
					Operation: "Read OpenShift cluster",
				},
				Origin: "user,system",
			},
			{
				Name: "Microsoft.RedHatOpenShift/openShiftClusters/write",
				Display: api.Display{
					Provider:  "Azure Red Hat OpenShift",
					Resource:  "openShiftClusters",
					Operation: "Write OpenShift cluster",
				},
				Origin: "user,system",
			},
			{
				Name: "Microsoft.RedHatOpenShift/openShiftClusters/delete",
				Display: api.Display{
					Provider:  "Azure Red Hat OpenShift",
					Resource:  "openShiftClusters",
					Operation: "Delete OpenShift cluster",
				},
				Origin: "user,system",
			},
			{
				Name: "Microsoft.RedHatOpenShift/openShiftClusters/listCredentials/action",
				Display: api.Display{
					Provider:  "Azure Red Hat OpenShift",
					Resource:  "openShiftClusters",
					Operation: "List credentials of an OpenShift cluster",
				},
				Origin: "user,system",
			},
			{
				Name: "Microsoft.RedHatOpenShift/openShiftClusters/listAdminCredentials/action",
				Display: api.Display{
					Provider:  "Azure Red Hat OpenShift",
					Resource:  "openShiftClusters",
					Operation: "List Admin Kubeconfig of an OpenShift cluster",
				},
				Origin: "user,system",
			},
			{
				Name: "Microsoft.RedHatOpenShift/operations/read",
				Display: api.Display{
					Provider:  "Azure Red Hat OpenShift",
					Resource:  "operations",
					Operation: "Read operations",
				},
				Origin: "user,system",
			},
		},
	}

	b, err := json.MarshalIndent(l, "", "    ")
	reply(log, w, nil, b, err)
}
