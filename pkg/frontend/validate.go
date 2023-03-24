package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	utilnamespace "github.com/Azure/ARO-RP/pkg/util/namespace"
	"github.com/Azure/ARO-RP/pkg/util/version"
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

func (f *frontend) validateSubscriptionState(ctx context.Context, path string, allowedStates ...api.SubscriptionState) (*api.SubscriptionDocument, error) {
	doc, err := f.getSubscriptionDocument(ctx, path)
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

func validatePermittedClusterwideObjects(gvr *schema.GroupVersionResource) bool {
	permittedGroups := map[string]bool{
		"apiserver.openshift.io":              true,
		"aro.openshift.io":                    true,
		"authorization.openshift.io":          true,
		"certificates.k8s.io":                 true,
		"config.openshift.io":                 true,
		"console.openshift.io":                true,
		"imageregistry.operator.openshift.io": true,
		"machine.openshift.io":                true,
		"machineconfiguration.openshift.io":   true,
		"operator.openshift.io":               true,
		"rbac.authorization.k8s.io":           true,
	}
	permittedObjects := map[string]map[string]bool{
		"": {"nodes": true},
	}
	allowedResources, groupHasException := permittedObjects[gvr.Group]
	return permittedGroups[gvr.Group] || (groupHasException && allowedResources[gvr.Resource])
}

func validateAdminKubernetesObjectsNonCustomer(method string, gvr *schema.GroupVersionResource, namespace, name string) error {
	if gvr == nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided resource is invalid.")
	}

	if namespace == "" && !validatePermittedClusterwideObjects(gvr) {
		return api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Access to cluster-scoped object '%v' is forbidden.", gvr)
	}

	if !utilnamespace.IsOpenShiftNamespace(namespace) {
		return api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Access to the provided namespace '%s' is forbidden.", namespace)
	}

	return validateAdminKubernetesObjects(method, gvr, namespace, name)
}

func validateAdminKubernetesObjects(method string, gvr *schema.GroupVersionResource, namespace, name string) error {
	if gvr == nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided resource is invalid.")
	}

	if gvr.Resource == "secrets" ||
		gvr.Group == "oauth.openshift.io" {
		return api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Access to secrets is forbidden.")
	}
	if method != http.MethodGet &&
		(gvr.Group == "rbac.authorization.k8s.io" ||
			gvr.Group == "authorization.openshift.io") {
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

func validateAdminKubernetesObjectsForceDelete(groupKind string) error {
	if !strings.EqualFold(groupKind, "Pod") {
		return api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Force deleting groupKind '%s' is forbidden.", groupKind)
	}

	return nil
}

func validateAdminVMName(vmName string) error {
	if vmName == "" || !rxKubernetesString.MatchString(vmName) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided vmName '%s' is invalid.", vmName)
	}

	return nil
}

func validateAdminKubernetesPodLogs(namespace, podName, containerName string) error {
	if podName == "" || !rxKubernetesString.MatchString(podName) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided pod name '%s' is invalid.", podName)
	}

	if namespace == "" || !rxKubernetesString.MatchString(namespace) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided namespace '%s' is invalid.", namespace)
	}
	// Checking if the namespace is an OpenShift namespace not a customer workload namespace.
	if !utilnamespace.IsOpenShiftNamespace(namespace) {
		return api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Access to the provided namespace '%s' is forbidden.", namespace)
	}

	if containerName == "" || !rxKubernetesString.MatchString(containerName) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided container name '%s' is invalid.", containerName)
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

func validateAdminMasterVMSize(vmSize string) error {
	// check to ensure that the target size is supported as a master size
	for k := range validate.SupportedMasterVmSizes {
		if strings.EqualFold(string(k), vmSize) {
			return nil
		}
	}

	return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided vmSize '%s' is unsupported for master.", vmSize)
}

// validateInstallVersion validates the install version set in the clusterprofile.version
// TODO convert this into static validation instead of this receiver function in the validation for frontend.
func (f *frontend) validateInstallVersion(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	oc := doc.OpenShiftCluster
	// If this request is from an older API or the user never specified
	// the version to install we default to the DefaultInstallStream.Version
	if oc.Properties.ClusterProfile.Version == "" {
		oc.Properties.ClusterProfile.Version = version.DefaultInstallStream.Version.String()
		return nil
	}

	f.mu.RLock()
	// we add the default installation version to the enabled ocp versions so no special case
	_, ok := f.enabledOcpVersions[oc.Properties.ClusterProfile.Version]
	f.mu.RUnlock()

	if !ok || !validate.RxInstallVersion.MatchString(oc.Properties.ClusterProfile.Version) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.clusterProfile.version", "The requested OpenShift version '%s' is invalid.", oc.Properties.ClusterProfile.Version)
	}

	return nil
}
