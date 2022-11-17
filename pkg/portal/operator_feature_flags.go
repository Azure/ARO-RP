package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/gorilla/mux"
)

func (p *portal) operatorFeatureFlags(w http.ResponseWriter, r *http.Request) {
	apiVars := mux.Vars(r)
	if apiVars == nil {
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
		return
	}

	subscription := apiVars["subscription"]
	resourceGroup := apiVars["resourceGroup"]
	clusterName := apiVars["name"]

	resourceId := strings.ToLower(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s", subscription, resourceGroup, clusterName))

	ctx := r.Context()
	clusterDoc, err := p.dbOpenShiftClusters.Get(ctx, resourceId)
	if err != nil {
		http.Error(w, "Cluster not found", http.StatusNotFound)
		return
	}

	operatorFlags := clusterDoc.OpenShiftCluster.Properties.OperatorFlags

	res, err := json.MarshalIndent(operatorFlags, "", "    ")
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(res)
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
		return
	}
}
