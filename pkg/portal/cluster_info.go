package portal

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"net/http"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/gorilla/mux"
)

type AdminOpenShiftClusterDocument struct {
	ResourceID              string               `json:"resourceId"`
	Name                    string               `json:"name"`
	Location                string               `json:"location"`
	CreatedBy               string               `json:"createdBy"`
	CreatedAt               string               `json:"createdAt"`
	LastModifiedBy          string               `json:"lastModifiedBy"`
	LastModifiedAt          string               `json:"lastModifiedAt"`
	Tags                    map[string]string    `json:"tags"`
	ArchitectureVersion     string               `json:"architectureVersion"`
	ProvisioningState       string               `json:"provisioningState"`
	LastProvisioningState   string               `json:"lastProvisioningState"`
	FailedProvisioningState string               `json:"failedProvisioningState"`
	LastAdminUpdateError    string               `json:"lastAdminUpdateError"`
	Version                 string               `json:"version"`
	ConsoleLink             string               `json:"consoleLink"`
	InfraId                 string               `json:"infraId"`
	MasterProfile           api.MasterProfile    `json:"masterProfile,omitempty"`
	WorkerProfiles          []api.WorkerProfile  `json:"workerProfile,omitempty"`
	ApiServerProfile        api.APIServerProfile `json:"apiServer,omitempty"`
	IngressProfiles         []api.IngressProfile `json:"ingressProfiles,omitempty"`
	Install                 *api.Install         `json:"install,omitempty"`
}

func (p *portal) clusterInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	apiVars := mux.Vars(r)

	subscription := apiVars["subscription"]
	resourceGroup := apiVars["resourceGroup"]
	clusterName := apiVars["name"]

	resourceId := "/subscriptions/" + subscription + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.RedHatOpenShift/openShiftClusters/" + clusterName

	doc, err := p.dbOpenShiftClusters.Get(ctx, resourceId)

	createdAt := "Unknown"
	if doc.OpenShiftCluster.SystemData.CreatedAt != nil {
		createdAt = doc.OpenShiftCluster.SystemData.CreatedAt.Format("2006-01-02 15:04:05")
	}

	lastModifiedAt := "Unknown"
	if doc.OpenShiftCluster.SystemData.CreatedAt != nil {
		lastModifiedAt = doc.OpenShiftCluster.SystemData.LastModifiedAt.Format("2006-01-02 15:04:05")
	}

	clusterInfo := AdminOpenShiftClusterDocument{
		ResourceID:              resourceId,
		Name:                    clusterName,
		Location:                doc.OpenShiftCluster.Location,
		CreatedBy:               doc.OpenShiftCluster.SystemData.CreatedBy,
		CreatedAt:               createdAt,
		LastModifiedBy:          doc.OpenShiftCluster.SystemData.LastModifiedBy,
		LastModifiedAt:          lastModifiedAt,
		Tags:                    doc.OpenShiftCluster.Tags,
		ArchitectureVersion:     string(rune(doc.OpenShiftCluster.Properties.ArchitectureVersion)),
		ProvisioningState:       doc.OpenShiftCluster.Properties.ProvisioningState.String(),
		LastProvisioningState:   doc.OpenShiftCluster.Properties.LastProvisioningState.String(),
		FailedProvisioningState: doc.OpenShiftCluster.Properties.FailedProvisioningState.String(),
		LastAdminUpdateError:    doc.OpenShiftCluster.Properties.LastAdminUpdateError,
		Version:                 doc.OpenShiftCluster.Properties.ClusterProfile.Version,
		ConsoleLink:             doc.OpenShiftCluster.Properties.ConsoleProfile.URL,
		InfraId:                 doc.OpenShiftCluster.Properties.InfraID,
		MasterProfile:           doc.OpenShiftCluster.Properties.MasterProfile,
		WorkerProfiles:          doc.OpenShiftCluster.Properties.WorkerProfiles,
		ApiServerProfile:        doc.OpenShiftCluster.Properties.APIServerProfile,
		IngressProfiles:         doc.OpenShiftCluster.Properties.IngressProfiles,
		Install:                 doc.OpenShiftCluster.Properties.Install,
	}

	b, err := json.MarshalIndent(clusterInfo, "", "    ")
	if err != nil {
		p.internalServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
}
