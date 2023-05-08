package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type AdminOpenShiftClusterDetail struct {
	Name                    string `json:"name"`
	Subscription            string `json:"subscription"`
	ResourceGroup           string `json:"resourceGroup"`
	ResourceId              string `json:"resourceId"`
	ProvisioningState       string `json:"provisioningState"`
	FailedProvisioningState string `json:"failedProvisioningState"`
	Version                 string `json:"version"`
	CreatedAt               string `json:"createdAt"`
	ProvisionedBy           string `json:"provisionedBy"`
	CreatedBy               string `json:"createdBy"`
	ArchitectureVersion     string `json:"architectureVersion"`
	LastProvisioningState   string `json:"lastProvisioningState"`
	LastAdminUpdateError    string `json:"lastAdminUpdateError"`
	InfraId                 string `json:"infraId"`
	ApiServerVisibility     string `json:"apiServerVisibility"`
	InstallPhase            string `json:"installStatus"`
}

func (p *portal) clusterInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	apiVars := mux.Vars(r)
	subscription := apiVars["subscription"]
	resourceGroup := apiVars["resourceGroup"]
	clusterName := apiVars["clusterName"]
	resourceId := p.getResourceID(subscription, resourceGroup, clusterName)

	doc, err := p.dbOpenShiftClusters.Get(ctx, resourceId)
	if err != nil {
		http.Error(w, "Cluster not found", http.StatusNotFound)
		return
	}

	createdAt := "Unknown"
	if !doc.OpenShiftCluster.Properties.CreatedAt.IsZero() {
		createdAt = doc.OpenShiftCluster.Properties.CreatedAt.Format(time.RFC3339)
	}

	installPhase := "Installed"
	if doc.OpenShiftCluster.Properties.Install != nil {
		installPhase = doc.OpenShiftCluster.Properties.Install.Phase.String()
	}

	clusterInfo := AdminOpenShiftClusterDetail{
		ResourceId:              resourceId,
		Name:                    clusterName,
		Subscription:            subscription,
		ResourceGroup:           resourceGroup,
		CreatedAt:               createdAt,
		ProvisionedBy:           doc.OpenShiftCluster.Properties.ProvisionedBy,
		ProvisioningState:       doc.OpenShiftCluster.Properties.ProvisioningState.String(),
		FailedProvisioningState: doc.OpenShiftCluster.Properties.FailedProvisioningState.String(),
		Version:                 doc.OpenShiftCluster.Properties.ClusterProfile.Version,

		CreatedBy:             doc.OpenShiftCluster.Properties.CreatedBy,
		ArchitectureVersion:   strconv.Itoa(int(doc.OpenShiftCluster.Properties.ArchitectureVersion)),
		LastProvisioningState: doc.OpenShiftCluster.Properties.LastProvisioningState.String(),
		LastAdminUpdateError:  doc.OpenShiftCluster.Properties.LastAdminUpdateError,
		InfraId:               doc.OpenShiftCluster.Properties.InfraID,
		ApiServerVisibility:   string(doc.OpenShiftCluster.Properties.APIServerProfile.Visibility),
		InstallPhase:          installPhase,
	}

	b, err := json.MarshalIndent(clusterInfo, "", "    ")
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}
