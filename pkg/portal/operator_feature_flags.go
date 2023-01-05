package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"gotest.tools/gotestsum/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
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
		log.Error(err.Error())
		return
	}

	restConfig, err := restconfig.RestConfig(p.dialer, clusterDoc.OpenShiftCluster)
	if err != nil {
		log.Error(err.Error())
		return
	}

	aroCli, err := aroclient.NewForConfig(restConfig)
	if err != nil {
		log.Error(err.Error())
		return
	}

	instance, err := aroCli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		http.Error(w, "Cluster not found", http.StatusNotFound)
		return
	}

	featureFlags := instance.Spec.OperatorFlags
	if featureFlags == nil {
		http.Error(w, "Error retriving operator flags", http.StatusNotFound)
	}

	res, err := json.MarshalIndent(featureFlags, "", "    ")
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
