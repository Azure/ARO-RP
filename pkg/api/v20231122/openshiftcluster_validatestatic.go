package v20231122

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/util/immutable"
	"github.com/Azure/ARO-RP/pkg/api/util/pullsecret"
	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	"github.com/Azure/ARO-RP/pkg/api/util/uuid"
	"github.com/Azure/ARO-RP/pkg/api/validate"
)

type openShiftClusterStaticValidator struct {
	location          string
	domain            string
	requireD2sWorkers bool
	resourceID        string

	r azure.Resource
}

// Validate validates an OpenShift cluster
func (sv openShiftClusterStaticValidator) Static(_oc interface{}, _current *api.OpenShiftCluster, location, domain string, requireD2sWorkers bool, installArchitectureVersion api.ArchitectureVersion, resourceID string) error {
	sv.location = location
	sv.domain = domain
	sv.requireD2sWorkers = requireD2sWorkers
	sv.resourceID = resourceID
	architectureVersion := installArchitectureVersion

	oc := _oc.(*OpenShiftCluster)

	var current *OpenShiftCluster
	if _current != nil {
		architectureVersion = _current.Properties.ArchitectureVersion
		current = (&openShiftClusterConverter{}).ToExternal(_current).(*OpenShiftCluster)
	}

	var err error
	sv.r, err = azure.ParseResourceID(sv.resourceID)
	if err != nil {
		return err
	}

	err = sv.validate(oc, current == nil, architectureVersion)
	if err != nil {
		return err
	}

	if current == nil {
		return nil
	}

	return sv.validateDelta(oc, current)
}

func (sv openShiftClusterStaticValidator) validate(oc *OpenShiftCluster, isCreate bool, architectureVersion api.ArchitectureVersion) error {
	if !strings.EqualFold(oc.ID, sv.resourceID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeMismatchingResourceID, "id", fmt.Sprintf("The provided resource ID '%s' did not match the name in the Url '%s'.", oc.ID, sv.resourceID))
	}
	if !strings.EqualFold(oc.Name, sv.r.ResourceName) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeMismatchingResourceName, "name", fmt.Sprintf("The provided resource name '%s' did not match the name in the Url '%s'.", oc.Name, sv.r.ResourceName))
	}
	if !strings.EqualFold(oc.Type, resourceProviderNamespace+"/"+resourceType) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeMismatchingResourceType, "type", fmt.Sprintf("The provided resource type '%s' did not match the name in the Url '%s'.", oc.Type, resourceProviderNamespace+"/"+resourceType))
	}
	if !strings.EqualFold(oc.Location, sv.location) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "location", fmt.Sprintf("The provided location '%s' is invalid.", oc.Location))
	}

	return sv.validateProperties("properties", &oc.Properties, isCreate, architectureVersion)
}

func (sv openShiftClusterStaticValidator) validateProperties(path string, p *OpenShiftClusterProperties, isCreate bool, architectureVersion api.ArchitectureVersion) error {
	switch p.ProvisioningState {
	case ProvisioningStateCreating, ProvisioningStateUpdating,
		ProvisioningStateAdminUpdating, ProvisioningStateDeleting,
		ProvisioningStateSucceeded, ProvisioningStateFailed, ProvisioningStateCanceled:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".provisioningState", fmt.Sprintf("The provided provisioning state '%s' is invalid.", p.ProvisioningState))
	}
	if err := sv.validateClusterProfile(path+".clusterProfile", &p.ClusterProfile, isCreate); err != nil {
		return err
	}
	if err := sv.validateConsoleProfile(path+".consoleProfile", &p.ConsoleProfile); err != nil {
		return err
	}
	if err := sv.validateServicePrincipalProfile(path+".servicePrincipalProfile", p.ServicePrincipalProfile); err != nil {
		return err
	}
	if err := sv.validateNetworkProfile(path+".networkProfile", &p.NetworkProfile, p.APIServerProfile.Visibility, p.IngressProfiles[0].Visibility); err != nil {
		return err
	}
	if err := sv.validateLoadBalancerProfile(path+".networkProfile.loadBalancerProfile", p.NetworkProfile.LoadBalancerProfile, isCreate, architectureVersion); err != nil {
		return err
	}
	if err := sv.validateMasterProfile(path+".masterProfile", &p.MasterProfile); err != nil {
		return err
	}
	if err := sv.validateAPIServerProfile(path+".apiserverProfile", &p.APIServerProfile); err != nil {
		return err
	}

	if isCreate {
		if len(p.WorkerProfilesStatus) != 0 {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".workerProfilesStatus", "Worker Profile Status must be set to nil.")
		}

		if len(p.WorkerProfiles) != 1 {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".workerProfiles", "There should be exactly one worker profile.")
		}
		if err := sv.validateWorkerProfile(path+".workerProfiles['"+p.WorkerProfiles[0].Name+"']", &p.WorkerProfiles[0], &p.MasterProfile); err != nil {
			return err
		}

		if len(p.IngressProfiles) != 1 {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".ingressProfiles", "There should be exactly one ingress profile.")
		}
		if err := sv.validateIngressProfile(path+".ingressProfiles['"+p.IngressProfiles[0].Name+"']", &p.IngressProfiles[0]); err != nil {
			return err
		}
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateClusterProfile(path string, cp *ClusterProfile, isCreate bool) error {
	if pullsecret.Validate(cp.PullSecret) != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".pullSecret", "The provided pull secret is invalid.")
	}
	if isCreate {
		if !validate.RxDomainName.MatchString(cp.Domain) {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".domain", fmt.Sprintf("The provided domain '%s' is invalid.", cp.Domain))
		}
	} else {
		// We currently do not allow domains with a digit as a first charecter,
		// for new clusters, but we already have some existing clusters with
		// domains like this and we need to allow customers to update them.
		if !validate.RxDomainNameRFC1123.MatchString(cp.Domain) {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".domain", fmt.Sprintf("The provided domain '%s' is invalid.", cp.Domain))
		}
	}
	// domain ends .aroapp.io, but doesn't end .<rp-location>.aroapp.io
	if strings.HasSuffix(cp.Domain, "."+strings.SplitN(sv.domain, ".", 2)[1]) &&
		!strings.HasSuffix(cp.Domain, "."+sv.domain) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".domain", fmt.Sprintf("The provided domain '%s' is invalid.", cp.Domain))
	}
	// domain is of form multiple.names.<rp-location>.aroapp.io
	if strings.HasSuffix(cp.Domain, "."+sv.domain) &&
		strings.ContainsRune(strings.TrimSuffix(cp.Domain, "."+sv.domain), '.') {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".domain", fmt.Sprintf("The provided domain '%s' is invalid.", cp.Domain))
	}

	if !validate.RxResourceGroupID.MatchString(cp.ResourceGroupID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".resourceGroupId", fmt.Sprintf("The provided resource group '%s' is invalid.", cp.ResourceGroupID))
	}
	if strings.Split(cp.ResourceGroupID, "/")[2] != sv.r.SubscriptionID {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".resourceGroupId", fmt.Sprintf("The provided resource group '%s' is invalid: must be in same subscription as cluster.", cp.ResourceGroupID))
	}
	if strings.EqualFold(cp.ResourceGroupID, fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", sv.r.SubscriptionID, sv.r.ResourceGroup)) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".resourceGroupId", fmt.Sprintf("The provided resource group '%s' is invalid: must be different from resourceGroup of the OpenShift cluster object.", cp.ResourceGroupID))
	}

	switch cp.FipsValidatedModules {
	case FipsValidatedModulesDisabled, FipsValidatedModulesEnabled:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".fipsValidatedModules", fmt.Sprintf("The provided value '%s' is invalid.", cp.FipsValidatedModules))
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateConsoleProfile(path string, cp *ConsoleProfile) error {
	if cp.URL != "" {
		if _, err := url.Parse(cp.URL); err != nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".url", fmt.Sprintf("The provided console URL '%s' is invalid.", cp.URL))
		}
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateServicePrincipalProfile(path string, spp *ServicePrincipalProfile) error {
	if spp == nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".servicePrincipalProfile", "ServicePrincipalProfile cannot be nil in this API version.")
	}

	valid := uuid.IsValid(spp.ClientID)
	if !valid {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".clientId", fmt.Sprintf("The provided client ID '%s' is invalid.", spp.ClientID))
	}
	if spp.ClientSecret == "" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".clientSecret", "The provided client secret is invalid.")
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateNetworkProfile(path string, np *NetworkProfile, apiServerVisibility Visibility, ingressVisibility Visibility) error {
	podIP, pod, err := net.ParseCIDR(np.PodCIDR)
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".podCidr", fmt.Sprintf("The provided pod CIDR '%s' is invalid: '%s'.", np.PodCIDR, err))
	}

	if pod.IP.To4() == nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".podCidr", fmt.Sprintf("The provided pod CIDR '%s' is invalid: must be IPv4.", np.PodCIDR))
	}

	for _, s := range api.JoinCIDRRange {
		_, cidr, _ := net.ParseCIDR(s)
		if cidr.Contains(pod.IP) || pod.Contains(cidr.IP) {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidCIDRRange, path, fmt.Sprintf("Azure Red Hat OpenShift uses 100.64.0.0/16, 169.254.169.0/29, and 100.88.0.0/16 IP address ranges internally. Do not include this '%s' IP address range in any other CIDR definitions in your cluster.", np.PodCIDR))
		}
	}

	ones, _ := pod.Mask.Size()
	if ones > 18 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".podCidr", fmt.Sprintf("The provided vnet CIDR '%s' is invalid: must be /18 or larger.", np.PodCIDR))
	}

	nip := podIP.Mask(pod.Mask)

	if nip.String() != podIP.String() {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidNetworkAddress, path+".podCidr", fmt.Sprintf("The provided pod CIDR '%s' is invalid, expecting: '%s/%d'.", np.PodCIDR, nip.String(), ones))
	}

	serviceIP, service, err := net.ParseCIDR(np.ServiceCIDR)
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".serviceCidr", fmt.Sprintf("The provided service CIDR '%s' is invalid: '%s'.", np.ServiceCIDR, err))
	}

	if service.IP.To4() == nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".serviceCidr", fmt.Sprintf("The provided service CIDR '%s' is invalid: must be IPv4.", np.ServiceCIDR))
	}

	for _, s := range api.JoinCIDRRange {
		_, cidr, _ := net.ParseCIDR(s)
		if cidr.Contains(service.IP) || service.Contains(cidr.IP) {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidCIDRRange, path, fmt.Sprintf("Azure Red Hat OpenShift uses 100.64.0.0/16, 169.254.169.0/29, and 100.88.0.0/16 IP address ranges internally. Do not include this '%s' IP address range in any other CIDR definitions in your cluster.", np.ServiceCIDR))
		}
	}

	ones, _ = service.Mask.Size()
	if ones > 22 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".serviceCidr", fmt.Sprintf("The provided vnet CIDR '%s' is invalid: must be /22 or larger.", np.ServiceCIDR))
	}

	nip = serviceIP.Mask(service.Mask)

	if nip.String() != serviceIP.String() {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidNetworkAddress, path+".serviceCidr", fmt.Sprintf("The provided service CIDR '%s' is invalid, expecting: '%s/%d'.", np.ServiceCIDR, nip.String(), ones))
	}

	if np.OutboundType != "" {
		if np.OutboundType != OutboundTypeLoadbalancer && np.OutboundType != OutboundTypeUserDefinedRouting {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".outboundType", fmt.Sprintf("The provided outboundType '%s' is invalid: must be UserDefinedRouting or Loadbalancer.", np.OutboundType))
		}
		if np.OutboundType == OutboundTypeUserDefinedRouting && (apiServerVisibility != VisibilityPrivate || ingressVisibility != VisibilityPrivate) {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".outboundType", fmt.Sprintf("The provided outboundType '%s' is invalid: cannot use UserDefinedRouting if either API Server Visibility or Ingress Visibility is public.", np.OutboundType))
		}
	}

	if np.OutboundType == OutboundTypeUserDefinedRouting && np.LoadBalancerProfile != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".loadBalancerProfile", "The provided loadBalancerProfile is invalid: cannot use a loadBalancerProfile if outboundType is UserDefinedRouting.")
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateLoadBalancerProfile(path string, lbp *LoadBalancerProfile, isCreate bool, architectureVersion api.ArchitectureVersion) error {
	if lbp == nil {
		return nil
	}

	switch {
	case lbp.ManagedOutboundIPs != nil:
		err := validateManagedOutboundIPs(path, *lbp.ManagedOutboundIPs, architectureVersion)
		if err != nil {
			return err
		}
	}
	// Prevents EffectiveOutboundIPs from being set during create,
	// during update validateDelta will prevent the field from being changed.
	if lbp.EffectiveOutboundIPs != nil && isCreate {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".effectiveOutboundIps", "The field effectiveOutboundIps is read only.")
	}
	return nil
}

func validateManagedOutboundIPs(path string, managedOutboundIPs ManagedOutboundIPs, architectureVersion api.ArchitectureVersion) error {
	if architectureVersion == api.ArchitectureVersionV1 && managedOutboundIPs.Count > 1 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".managedOutboundIps.count", fmt.Sprintf("The provided managedOutboundIps.count %d is invalid: managedOutboundIps.count must be 1, multiple IPs are not supported for this cluster's network architecture.", managedOutboundIPs.Count))
	}
	if !(managedOutboundIPs.Count > 0 && managedOutboundIPs.Count <= 20) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".managedOutboundIps.count", fmt.Sprintf("The provided managedOutboundIps.count %d is invalid: managedOutboundIps.count must be in the range of 1 to 20 (inclusive).", managedOutboundIPs.Count))
	}
	return nil
}

func (sv openShiftClusterStaticValidator) validateMasterProfile(path string, mp *MasterProfile) error {
	if !validate.VMSizeIsValid(api.VMSize(mp.VMSize), sv.requireD2sWorkers, true) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".vmSize", fmt.Sprintf("The provided master VM size '%s' is invalid.", mp.VMSize))
	}
	if !validate.RxSubnetID.MatchString(mp.SubnetID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", fmt.Sprintf("The provided master VM subnet '%s' is invalid.", mp.SubnetID))
	}
	sr, err := azure.ParseResourceID(mp.SubnetID)
	if err != nil {
		return err
	}
	if sr.SubscriptionID != sv.r.SubscriptionID {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", fmt.Sprintf("The provided master VM subnet '%s' is invalid: must be in same subscription as cluster.", mp.SubnetID))
	}
	switch mp.EncryptionAtHost {
	case EncryptionAtHostDisabled, EncryptionAtHostEnabled:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".encryptionAtHost", fmt.Sprintf("The provided value '%s' is invalid.", mp.EncryptionAtHost))
	}
	if mp.DiskEncryptionSetID != "" {
		if !validate.RxDiskEncryptionSetID.MatchString(mp.DiskEncryptionSetID) {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".diskEncryptionSetId", fmt.Sprintf("The provided master disk encryption set '%s' is invalid.", mp.DiskEncryptionSetID))
		}
		desr, err := azure.ParseResourceID(mp.DiskEncryptionSetID)
		if err != nil {
			return err
		}
		if desr.SubscriptionID != sv.r.SubscriptionID {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".diskEncryptionSetId", fmt.Sprintf("The provided master disk encryption set '%s' is invalid: must be in same subscription as cluster.", mp.DiskEncryptionSetID))
		}
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateWorkerProfile(path string, wp *WorkerProfile, mp *MasterProfile) error {
	if wp.Name != "worker" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".name", fmt.Sprintf("The provided worker name '%s' is invalid.", wp.Name))
	}
	if !validate.VMSizeIsValid(api.VMSize(wp.VMSize), sv.requireD2sWorkers, false) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".vmSize", fmt.Sprintf("The provided worker VM size '%s' is invalid.", wp.VMSize))
	}
	if !validate.DiskSizeIsValid(wp.DiskSizeGB) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".diskSizeGB", fmt.Sprintf("The provided worker disk size '%d' is invalid.", wp.DiskSizeGB))
	}
	if !validate.RxSubnetID.MatchString(wp.SubnetID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", fmt.Sprintf("The provided worker VM subnet '%s' is invalid.", wp.SubnetID))
	}
	switch wp.EncryptionAtHost {
	case EncryptionAtHostDisabled, EncryptionAtHostEnabled:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".encryptionAtHost", fmt.Sprintf("The provided value '%s' is invalid.", wp.EncryptionAtHost))
	}
	workerVnetID, _, err := apisubnet.Split(wp.SubnetID)
	if err != nil {
		return err
	}
	masterVnetID, _, err := apisubnet.Split(mp.SubnetID)
	if err != nil {
		return err
	}
	if !strings.EqualFold(masterVnetID, workerVnetID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", fmt.Sprintf("The provided worker VM subnet '%s' is invalid: must be in the same vnet as master VM subnet '%s'.", wp.SubnetID, mp.SubnetID))
	}
	if strings.EqualFold(mp.SubnetID, wp.SubnetID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", fmt.Sprintf("The provided worker VM subnet '%s' is invalid: must be different to master VM subnet '%s'.", wp.SubnetID, mp.SubnetID))
	}
	if wp.Count < 2 || wp.Count > 50 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".count", fmt.Sprintf("The provided worker count '%d' is invalid.", wp.Count))
	}
	if !strings.EqualFold(mp.DiskEncryptionSetID, wp.DiskEncryptionSetID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", fmt.Sprintf("The provided worker disk encryption set '%s' is invalid: must be the same as master disk encryption set '%s'.", wp.DiskEncryptionSetID, mp.DiskEncryptionSetID))
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateAPIServerProfile(path string, ap *APIServerProfile) error {
	switch ap.Visibility {
	case VisibilityPublic, VisibilityPrivate:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".visibility", fmt.Sprintf("The provided visibility '%s' is invalid.", ap.Visibility))
	}
	if ap.URL != "" {
		if _, err := url.Parse(ap.URL); err != nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".url", fmt.Sprintf("The provided URL '%s' is invalid.", ap.URL))
		}
	}
	if ap.IP != "" {
		ip := net.ParseIP(ap.IP)
		if ip == nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".ip", fmt.Sprintf("The provided IP '%s' is invalid.", ap.IP))
		}
		if ip.To4() == nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".ip", fmt.Sprintf("The provided IP '%s' is invalid: must be IPv4.", ap.IP))
		}
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateIngressProfile(path string, p *IngressProfile) error {
	if p.Name != "default" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".name", fmt.Sprintf("The provided ingress name '%s' is invalid.", p.Name))
	}
	switch p.Visibility {
	case VisibilityPublic, VisibilityPrivate:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".visibility", fmt.Sprintf("The provided visibility '%s' is invalid.", p.Visibility))
	}
	if p.IP != "" {
		ip := net.ParseIP(p.IP)
		if ip == nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".ip", fmt.Sprintf("The provided IP '%s' is invalid.", p.IP))
		}
		if ip.To4() == nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".ip", fmt.Sprintf("The provided IP '%s' is invalid: must be IPv4.", p.IP))
		}
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateDelta(oc, current *OpenShiftCluster) error {
	err := immutable.Validate("", oc, current)
	if err != nil {
		err := err.(*immutable.ValidationError)
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, err.Target, err.Message)
	}

	return nil
}
