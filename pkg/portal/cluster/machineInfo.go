package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/gorilla/mux"
)

// This is a generic API to return a slice the following VM information.
type vmDetails struct {
	Name             string `json:"name,omitempty"`
	AllocationStatus string `json:"allocationStatus,omitempty"`
	VMID             string `json:"vmid,omitempty"`
}

func (p *portal) machineInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	apiVars := mux.Vars(r)
	subscription := apiVars["subscription"]
	resourceGroup := apiVars["resourceGroup"]
	clusterName := apiVars["name"]
	resourceID := strings.ToLower(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s", subscription, resourceGroup, clusterName))

	doc, err := p.dbOpenShiftClusters.Get(ctx, resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		p.log.Error(api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "",
			"The Resource '%s/%s' under resource group '%s' was not found."))
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Initializing the cluster's Resource Group name.
	clusterRGName := stringutils.LastTokenByte(doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	subscriptionDoc, err := p.getSubscriptionDocument(ctx, doc.Key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fpAuth, err := p.env.FPAuthorizer(subscriptionDoc.Subscription.Properties.TenantID, p.env.Environment().ResourceManagerEndpoint)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Getting Virtual Machine resources through the Cluster's Resource Group
	computeResources, err := features.NewResourcesClient(p.env.Environment(), subscriptionDoc.ID, fpAuth).ListByResourceGroup(ctx, clusterRGName, "resourceType eq 'Microsoft.Compute/virtualMachines'", "", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	vmDetailList := make([]vmDetails, 0)
	//armResources := make([]arm.Resource, 0, len(computeResources))
	virtualMachineClient := compute.NewVirtualMachinesClient(p.env.Environment(), subscriptionDoc.ID, fpAuth)
	for _, res := range computeResources {
		var VMID, vmName, allocationStatus string
		if *res.Type != "Microsoft.Compute/virtualMachines" {
			continue
		}

		vm, err := virtualMachineClient.Get(ctx, clusterRGName, *res.Name, mgmtcompute.InstanceView)
		if err != nil {
			p.log.Warn(err) // can happen when the ARM cache is lagging
			// armResources = append(armResources, arm.Resource{
			// 	Resource: res,
			// })
			continue
		}

		vmName = *vm.Name
		VMID = *vm.VMID
		instanceViewStatuses := vm.InstanceView.Statuses
		for _, status := range *instanceViewStatuses {
			if strings.Contains(*status.Code, "PowerState") {
				allocationStatus = *status.Code
			}
		}

		vmDetailList = append(vmDetailList, vmDetails{
			Name:             vmName,
			AllocationStatus: allocationStatus,
			VMID:             VMID,
		})
	}
	vmDetailListJSON, err := json.MarshalIndent(vmDetailList, "", "	 ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Println(string(vmDetailListJSON))
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(vmDetailListJSON)
}

func (p *portal) getSubscriptionDocument(ctx context.Context, key string) (*api.SubscriptionDocument, error) {
	r, err := azure.ParseResourceID(key)
	if err != nil {
		return nil, err
	}
	doc, err := p.dbSubscriptions.Get(ctx, r.SubscriptionID)
	if cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidSubscriptionState, "", "Request is not allowed in unregistered subscription '%s'.", r.SubscriptionID)
	}

	return doc, err
}
