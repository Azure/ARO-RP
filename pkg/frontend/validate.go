package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	utilnamespace "github.com/Azure/ARO-RP/pkg/util/namespace"
)

func validateTerminalProvisioningState(state api.ProvisioningState) error {
	if state.IsTerminal() {
		return nil
	}

	return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeRequestNotAllowed, "", "Request is not allowed in provisioningState '%s'.", state)
}

func (f *frontend) getSubscriptionDocument(ctx context.Context, key string) (*api.SubscriptionDocument, error) {
	r, err := azure.ParseResourceID(key)
	if err != nil {
		return nil, err
	}

	doc, err := f.dbSubscriptions.Get(ctx, r.SubscriptionID)
	if cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidSubscriptionState, "", "Request is not allowed in unregistered subscription '%s'.", r.SubscriptionID)
	}

	return doc, err
}

func (f *frontend) validateSubscriptionState(ctx context.Context, key string, allowedStates ...api.SubscriptionState) (*api.SubscriptionDocument, error) {
	doc, err := f.getSubscriptionDocument(ctx, key)
	if err != nil {
		return nil, err
	}

	for _, allowedState := range allowedStates {
		if doc.Subscription.State == allowedState {
			return doc, nil
		}
	}

	return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidSubscriptionState, "", "Request is not allowed in subscription in state '%s'.", doc.Subscription.State)
}

// validateOpenShiftUniqueKey returns which unique key if causing a 412 error
func (f *frontend) validateOpenShiftUniqueKey(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	docs, err := f.dbOpenShiftClusters.GetByClientID(ctx, doc.PartitionKey, doc.ClientIDKey)
	if err != nil {
		return err
	}
	if docs.Count != 0 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeDuplicateClientID, "", "The provided client ID '%s' is already in use by a cluster.", doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientID)
	}
	docs, err = f.dbOpenShiftClusters.GetByClusterResourceGroupID(ctx, doc.PartitionKey, doc.ClusterResourceGroupIDKey)
	if err != nil {
		return err
	}
	if docs.Count != 0 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeDuplicateResourceGroup, "", "The provided resource group '%s' already contains a cluster.", doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID)
	}
	return api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
}

// rxKubernetesString is weaker than Kubernetes validation, but strong enough to
// prevent mischief
var rxKubernetesString = regexp.MustCompile(`(?i)^[-a-z0-9.]{0,255}$`)

func validateAdminKubernetesObjectsNonCustomer(method, groupKind, namespace, name string) error {
	if !utilnamespace.IsOpenShift(namespace) {
		return api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Access to the provided namespace '%s' is forbidden.", namespace)
	}

	return validateAdminKubernetesObjects(method, groupKind, namespace, name)
}

func validateAdminKubernetesObjects(method, groupKind, namespace, name string) error {
	if groupKind == "" ||
		!rxKubernetesString.MatchString(groupKind) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided groupKind '%s' is invalid.", groupKind)
	}
	if strings.EqualFold(groupKind, "Secret") ||
		strings.HasSuffix(strings.ToLower(groupKind), ".oauth.openshift.io") {
		return api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Access to secrets is forbidden.")
	}
	if method != http.MethodGet &&
		(strings.HasSuffix(strings.ToLower(groupKind), ".rbac.authorization.k8s.io") ||
			strings.HasSuffix(strings.ToLower(groupKind), ".authorization.openshift.io")) {
		return api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Write access to RBAC is forbidden.")
	}

	if !rxKubernetesString.MatchString(namespace) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided namespace '%s' is invalid.", namespace)
	}

	if (method != http.MethodGet && name == "") ||
		!rxKubernetesString.MatchString(name) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided name '%s' is invalid.", name)
	}

	return nil
}

func validateAdminVMName(vmName string) error {
	if vmName == "" || !rxKubernetesString.MatchString(vmName) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided vmName '%s' is invalid.", vmName)
	}

	return nil
}

// Azure resource name rules:
// https://docs.microsoft.com/en-us/azure/azure-resource-manager/management/resource-name-rules#microsoftnetwork
var rxNetworkInterfaceName = regexp.MustCompile(`^[a-zA-Z0-9].*\w$`)

func validateNetworkInterfaceName(nicName string) error {
	if nicName == "" || !rxNetworkInterfaceName.MatchString(nicName) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided nicName '%s' is invalid.", nicName)
	}
	return nil
}

func validateAdminVMSize(vmSize string) error {
	if vmSize == "" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided vmSize '%s' is invalid.", vmSize)
	}
	return nil
}
