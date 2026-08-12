package v20250725

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/coreos/go-semver/semver"

	azcorearm "github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/util/immutable"
	"github.com/Azure/ARO-RP/pkg/api/util/pullsecret"
	apisubnet "github.com/Azure/ARO-RP/pkg/api/util/subnet"
	"github.com/Azure/ARO-RP/pkg/api/util/uuid"
	"github.com/Azure/ARO-RP/pkg/api/v20250725/generated"
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
	if !strings.EqualFold(value(oc.ID), sv.resourceID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeMismatchingResourceID, "id", fmt.Sprintf("The provided resource ID '%s' did not match the name in the Url '%s'.", value(oc.ID), sv.resourceID))
	}
	if !strings.EqualFold(value(oc.Name), sv.r.ResourceName) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeMismatchingResourceName, "name", fmt.Sprintf("The provided resource name '%s' did not match the name in the Url '%s'.", value(oc.Name), sv.r.ResourceName))
	}
	if !strings.EqualFold(value(oc.Type), resourceProviderNamespace+"/"+resourceType) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeMismatchingResourceType, "type", fmt.Sprintf("The provided resource type '%s' did not match the name in the Url '%s'.", value(oc.Type), resourceProviderNamespace+"/"+resourceType))
	}
	if !strings.EqualFold(value(oc.Location), sv.location) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "location", fmt.Sprintf("The provided location '%s' is invalid.", value(oc.Location)))
	}

	if err := sv.validatePlatformIdentities(oc); err != nil {
		return err
	}

	return sv.validateProperties("properties", oc.Properties, isCreate, architectureVersion)
}

func (sv openShiftClusterStaticValidator) validateProperties(path string, p *generated.OpenShiftClusterProperties, isCreate bool, architectureVersion api.ArchitectureVersion) error {
	switch value(p.ProvisioningState) {
	case generated.ProvisioningStateCreating, generated.ProvisioningStateUpdating,
		generated.ProvisioningStateAdminUpdating, generated.ProvisioningStateDeleting,
		generated.ProvisioningStateSucceeded, generated.ProvisioningStateFailed, generated.ProvisioningStateCanceled:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".provisioningState", fmt.Sprintf("The provided provisioning state '%s' is invalid.", value(p.ProvisioningState)))
	}
	if err := sv.validateClusterProfile(path+".clusterProfile", p.ClusterProfile, isCreate); err != nil {
		return err
	}
	if err := sv.validateConsoleProfile(path+".consoleProfile", p.ConsoleProfile); err != nil {
		return err
	}
	if err := sv.validateServicePrincipalProfile(path+".servicePrincipalProfile", p.ServicePrincipalProfile); err != nil {
		return err
	}
	if len(p.IngressProfiles) > 0 {
		if err := sv.validateNetworkProfile(path+".networkProfile", p.NetworkProfile, value(p.ApiserverProfile.Visibility), value(p.IngressProfiles[0].Visibility), isCreate); err != nil {
			return err
		}
	}
	if err := sv.validateLoadBalancerProfile(path+".networkProfile.loadBalancerProfile", p.NetworkProfile.LoadBalancerProfile, isCreate, architectureVersion); err != nil {
		return err
	}
	if err := sv.validateMasterProfile(path+".masterProfile", p.MasterProfile, value(p.ClusterProfile.Version)); err != nil {
		return err
	}
	if err := sv.validateAPIServerProfile(path+".apiserverProfile", p.ApiserverProfile); err != nil {
		return err
	}
	if err := sv.validatePlatformWorkloadIdentityProfile(path+".platformWorkloadIdentityProfile", p.PlatformWorkloadIdentityProfile); err != nil {
		return err
	}

	if isCreate {
		if len(p.WorkerProfilesStatus) != 0 {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".workerProfilesStatus", "Worker Profile Status must be set to nil.")
		}

		if len(p.WorkerProfiles) != 1 {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".workerProfiles", "There should be exactly one worker profile.")
		}
		if err := sv.validateWorkerProfile(path+".workerProfiles['"+value(p.WorkerProfiles[0].Name)+"']", p.WorkerProfiles[0], p.MasterProfile, value(p.ClusterProfile.Version)); err != nil {
			return err
		}

		if len(p.IngressProfiles) != 1 {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".ingressProfiles", "There should be exactly one ingress profile.")
		}
		if err := sv.validateIngressProfile(path+".ingressProfiles['"+value(p.IngressProfiles[0].Name)+"']", p.IngressProfiles[0]); err != nil {
			return err
		}
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateClusterProfile(path string, cp *generated.ClusterProfile, isCreate bool) error {
	if pullsecret.Validate(value(cp.PullSecret)) != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".pullSecret", "The provided pull secret is invalid.")
	}
	if isCreate {
		if !validate.RxDomainName.MatchString(value(cp.Domain)) {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".domain", fmt.Sprintf("The provided domain '%s' is invalid.", value(cp.Domain)))
		}
	} else {
		// We currently do not allow domains with a digit as a first charecter,
		// for new clusters, but we already have some existing clusters with
		// domains like this and we need to allow customers to update them.
		if !validate.RxDomainNameRFC1123.MatchString(value(cp.Domain)) {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".domain", fmt.Sprintf("The provided domain '%s' is invalid.", value(cp.Domain)))
		}
	}
	// domain ends .aroapp.io, but doesn't end .<rp-location>.aroapp.io
	if strings.HasSuffix(value(cp.Domain), "."+strings.SplitN(sv.domain, ".", 2)[1]) &&
		!strings.HasSuffix(value(cp.Domain), "."+sv.domain) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".domain", fmt.Sprintf("The provided domain '%s' is invalid.", value(cp.Domain)))
	}
	// domain is of form multiple.names.<rp-location>.aroapp.io
	if strings.HasSuffix(value(cp.Domain), "."+sv.domain) &&
		strings.ContainsRune(strings.TrimSuffix(value(cp.Domain), "."+sv.domain), '.') {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".domain", fmt.Sprintf("The provided domain '%s' is invalid.", value(cp.Domain)))
	}

	if !validate.RxResourceGroupID.MatchString(value(cp.ResourceGroupID)) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".resourceGroupId", fmt.Sprintf("The provided resource group '%s' is invalid.", value(cp.ResourceGroupID)))
	}
	if strings.Split(value(cp.ResourceGroupID), "/")[2] != sv.r.SubscriptionID {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".resourceGroupId", fmt.Sprintf("The provided resource group '%s' is invalid: must be in same subscription as cluster.", value(cp.ResourceGroupID)))
	}
	if strings.EqualFold(value(cp.ResourceGroupID), fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", sv.r.SubscriptionID, sv.r.ResourceGroup)) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".resourceGroupId", fmt.Sprintf("The provided resource group '%s' is invalid: must be different from resourceGroup of the OpenShift cluster object.", value(cp.ResourceGroupID)))
	}

	switch value(cp.FipsValidatedModules) {
	case generated.FipsValidatedModulesDisabled, generated.FipsValidatedModulesEnabled:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".fipsValidatedModules", fmt.Sprintf("The provided value '%s' is invalid.", value(cp.FipsValidatedModules)))
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateConsoleProfile(path string, cp *generated.ConsoleProfile) error {
	if cp == nil {
		return nil
	}

	if value(cp.URL) != "" {
		if _, err := url.Parse(value(cp.URL)); err != nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".url", fmt.Sprintf("The provided console URL '%s' is invalid.", value(cp.URL)))
		}
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateServicePrincipalProfile(path string, spp *generated.ServicePrincipalProfile) error {
	if spp == nil {
		return nil
	}

	valid := uuid.IsValid(value(spp.ClientID))
	if !valid {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".clientId", fmt.Sprintf("The provided client ID '%s' is invalid.", value(spp.ClientID)))
	}
	if value(spp.ClientSecret) == "" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".clientSecret", "The provided client secret is invalid.")
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateNetworkProfile(path string, np *generated.NetworkProfile, apiServerVisibility generated.Visibility, ingressVisibility generated.Visibility, isCreate bool) error {
	podCIDR := value(np.PodCidr)
	serviceCIDR := value(np.ServiceCidr)
	outboundType := value(np.OutboundType)
	podIP, pod, err := net.ParseCIDR(podCIDR)
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".podCidr", fmt.Sprintf("The provided pod CIDR '%s' is invalid: '%s'.", podCIDR, err))
	}

	if pod.IP.To4() == nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".podCidr", fmt.Sprintf("The provided pod CIDR '%s' is invalid: must be IPv4.", podCIDR))
	}

	// Only validate against JoinCIDRRange during cluster creation
	// For existing clusters, allow OVN default ranges to support SDN->OVN migrations
	if isCreate {
		for _, s := range api.JoinCIDRRange {
			_, cidr, _ := net.ParseCIDR(s)
			if cidr.Contains(pod.IP) || pod.Contains(cidr.IP) {
				return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidCIDRRange, path, fmt.Sprintf("Azure Red Hat OpenShift uses 100.64.0.0/16, 169.254.169.0/29, and 100.88.0.0/16 IP address ranges internally. Do not include this '%s' IP address range in any other CIDR definitions in your cluster.", podCIDR))
			}
		}
	}

	ones, _ := pod.Mask.Size()
	if ones > 18 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".podCidr", fmt.Sprintf("The provided vnet CIDR '%s' is invalid: must be /18 or larger.", podCIDR))
	}

	nip := podIP.Mask(pod.Mask)

	if nip.String() != podIP.String() {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidNetworkAddress, path+".podCidr", fmt.Sprintf("The provided pod CIDR '%s' is invalid, expecting: '%s/%d'.", podCIDR, nip.String(), ones))
	}

	serviceIP, service, err := net.ParseCIDR(serviceCIDR)
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".serviceCidr", fmt.Sprintf("The provided service CIDR '%s' is invalid: '%s'.", serviceCIDR, err))
	}

	if service.IP.To4() == nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".serviceCidr", fmt.Sprintf("The provided service CIDR '%s' is invalid: must be IPv4.", serviceCIDR))
	}

	// Only validate against JoinCIDRRange during cluster creation
	// For existing clusters, allow OVN default ranges to support SDN->OVN migrations
	if isCreate {
		for _, s := range api.JoinCIDRRange {
			_, cidr, _ := net.ParseCIDR(s)
			if cidr.Contains(service.IP) || service.Contains(cidr.IP) {
				return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidCIDRRange, path, fmt.Sprintf("Azure Red Hat OpenShift uses 100.64.0.0/16, 169.254.169.0/29, and 100.88.0.0/16 IP address ranges internally. Do not include this '%s' IP address range in any other CIDR definitions in your cluster.", serviceCIDR))
			}
		}
	}

	ones, _ = service.Mask.Size()
	if ones > 22 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".serviceCidr", fmt.Sprintf("The provided vnet CIDR '%s' is invalid: must be /22 or larger.", serviceCIDR))
	}

	nip = serviceIP.Mask(service.Mask)

	if nip.String() != serviceIP.String() {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidNetworkAddress, path+".serviceCidr", fmt.Sprintf("The provided service CIDR '%s' is invalid, expecting: '%s/%d'.", serviceCIDR, nip.String(), ones))
	}

	if outboundType != "" {
		if outboundType != generated.OutboundTypeLoadbalancer && outboundType != generated.OutboundTypeUserDefinedRouting {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".outboundType", fmt.Sprintf("The provided outboundType '%s' is invalid: must be UserDefinedRouting or Loadbalancer.", outboundType))
		}
		if outboundType == generated.OutboundTypeUserDefinedRouting && (apiServerVisibility != generated.VisibilityPrivate || ingressVisibility != generated.VisibilityPrivate) {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".outboundType", fmt.Sprintf("The provided outboundType '%s' is invalid: cannot use UserDefinedRouting if either API Server Visibility or Ingress Visibility is public.", outboundType))
		}
	}

	if outboundType == generated.OutboundTypeUserDefinedRouting && np.LoadBalancerProfile != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".loadBalancerProfile", "The provided loadBalancerProfile is invalid: cannot use a loadBalancerProfile if outboundType is UserDefinedRouting.")
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateLoadBalancerProfile(path string, lbp *generated.LoadBalancerProfile, isCreate bool, architectureVersion api.ArchitectureVersion) error {
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

func validateManagedOutboundIPs(path string, managedOutboundIPs generated.ManagedOutboundIPs, architectureVersion api.ArchitectureVersion) error {
	count := value(managedOutboundIPs.Count)
	if architectureVersion == api.ArchitectureVersionV1 && count > 1 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".managedOutboundIps.count", fmt.Sprintf("The provided managedOutboundIps.count %d is invalid: managedOutboundIps.count must be 1, multiple IPs are not supported for this cluster's network architecture.", count))
	}
	if count <= 0 || count > 20 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".managedOutboundIps.count", fmt.Sprintf("The provided managedOutboundIps.count %d is invalid: managedOutboundIps.count must be in the range of 1 to 20 (inclusive).", count))
	}
	return nil
}

func (sv openShiftClusterStaticValidator) validateMasterProfile(path string, mp *generated.MasterProfile, version string) error {
	switch validate.VMSizeIsValidForVersion(api.VMSize(value(mp.VMSize)), sv.requireD2sWorkers, true, version) {
	case validate.VMValidityNotSupportedForRole:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".vmSize", fmt.Sprintf("The provided VM size '%s' is invalid for the 'master' role.", value(mp.VMSize)))
	case validate.VMValidityNotSupportedInVersion:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".vmSize", fmt.Sprintf("The provided master VM size '%s' is invalid for the chosen OpenShift version.", value(mp.VMSize)))
	}
	if !validate.RxSubnetID.MatchString(value(mp.SubnetID)) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", fmt.Sprintf("The provided master VM subnet '%s' is invalid.", value(mp.SubnetID)))
	}
	sr, err := azure.ParseResourceID(value(mp.SubnetID))
	if err != nil {
		return err
	}
	if sr.SubscriptionID != sv.r.SubscriptionID {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", fmt.Sprintf("The provided master VM subnet '%s' is invalid: must be in same subscription as cluster.", value(mp.SubnetID)))
	}
	switch value(mp.EncryptionAtHost) {
	case generated.EncryptionAtHostDisabled, generated.EncryptionAtHostEnabled:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".encryptionAtHost", fmt.Sprintf("The provided value '%s' is invalid.", value(mp.EncryptionAtHost)))
	}
	if value(mp.DiskEncryptionSetID) != "" {
		if !validate.RxDiskEncryptionSetID.MatchString(value(mp.DiskEncryptionSetID)) {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".diskEncryptionSetId", fmt.Sprintf("The provided master disk encryption set '%s' is invalid.", value(mp.DiskEncryptionSetID)))
		}
		desr, err := azure.ParseResourceID(value(mp.DiskEncryptionSetID))
		if err != nil {
			return err
		}
		if desr.SubscriptionID != sv.r.SubscriptionID {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".diskEncryptionSetId", fmt.Sprintf("The provided master disk encryption set '%s' is invalid: must be in same subscription as cluster.", value(mp.DiskEncryptionSetID)))
		}
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateWorkerProfile(path string, wp *generated.WorkerProfile, mp *generated.MasterProfile, version string) error {
	if value(wp.Name) != "worker" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".name", fmt.Sprintf("The provided worker name '%s' is invalid.", value(wp.Name)))
	}
	switch validate.VMSizeIsValidForVersion(api.VMSize(value(wp.VMSize)), sv.requireD2sWorkers, false, version) {
	case validate.VMValidityNotSupportedForRole:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".vmSize", fmt.Sprintf("The provided VM size '%s' is invalid for the 'worker' role.", value(wp.VMSize)))
	case validate.VMValidityNotSupportedInVersion:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".vmSize", fmt.Sprintf("The provided worker VM size '%s' is invalid for the chosen OpenShift version.", value(wp.VMSize)))
	}
	if !validate.DiskSizeIsValid(int(value(wp.DiskSizeGB))) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".diskSizeGB", fmt.Sprintf("The provided worker disk size '%d' is invalid.", value(wp.DiskSizeGB)))
	}
	if !validate.RxSubnetID.MatchString(value(wp.SubnetID)) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", fmt.Sprintf("The provided worker VM subnet '%s' is invalid.", value(wp.SubnetID)))
	}
	switch value(wp.EncryptionAtHost) {
	case generated.EncryptionAtHostDisabled, generated.EncryptionAtHostEnabled:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".encryptionAtHost", fmt.Sprintf("The provided value '%s' is invalid.", value(wp.EncryptionAtHost)))
	}
	workerVnetID, _, err := apisubnet.Split(value(wp.SubnetID))
	if err != nil {
		return err
	}
	masterVnetID, _, err := apisubnet.Split(value(mp.SubnetID))
	if err != nil {
		return err
	}
	if !strings.EqualFold(masterVnetID, workerVnetID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", fmt.Sprintf("The provided worker VM subnet '%s' is invalid: must be in the same vnet as master VM subnet '%s'.", value(wp.SubnetID), value(mp.SubnetID)))
	}
	if strings.EqualFold(value(mp.SubnetID), value(wp.SubnetID)) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", fmt.Sprintf("The provided worker VM subnet '%s' is invalid: must be different to master VM subnet '%s'.", value(wp.SubnetID), value(mp.SubnetID)))
	}
	if value(wp.Count) < 2 || value(wp.Count) > 50 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".count", fmt.Sprintf("The provided worker count '%d' is invalid.", value(wp.Count)))
	}
	if !strings.EqualFold(value(mp.DiskEncryptionSetID), value(wp.DiskEncryptionSetID)) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", fmt.Sprintf("The provided worker disk encryption set '%s' is invalid: must be the same as master disk encryption set '%s'.", value(wp.DiskEncryptionSetID), value(mp.DiskEncryptionSetID)))
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateAPIServerProfile(path string, ap *generated.APIServerProfile) error {
	switch value(ap.Visibility) {
	case generated.VisibilityPublic, generated.VisibilityPrivate:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".visibility", fmt.Sprintf("The provided visibility '%s' is invalid.", value(ap.Visibility)))
	}
	if value(ap.URL) != "" {
		if _, err := url.Parse(value(ap.URL)); err != nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".url", fmt.Sprintf("The provided URL '%s' is invalid.", value(ap.URL)))
		}
	}
	if value(ap.IP) != "" {
		ip := net.ParseIP(value(ap.IP))
		if ip == nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".ip", fmt.Sprintf("The provided IP '%s' is invalid.", value(ap.IP)))
		}
		if ip.To4() == nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".ip", fmt.Sprintf("The provided IP '%s' is invalid: must be IPv4.", value(ap.IP)))
		}
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateIngressProfile(path string, p *generated.IngressProfile) error {
	if value(p.Name) != "default" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".name", fmt.Sprintf("The provided ingress name '%s' is invalid.", value(p.Name)))
	}
	switch value(p.Visibility) {
	case generated.VisibilityPublic, generated.VisibilityPrivate:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".visibility", fmt.Sprintf("The provided visibility '%s' is invalid.", value(p.Visibility)))
	}
	if value(p.IP) != "" {
		ip := net.ParseIP(value(p.IP))
		if ip == nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".ip", fmt.Sprintf("The provided IP '%s' is invalid.", value(p.IP)))
		}
		if ip.To4() == nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".ip", fmt.Sprintf("The provided IP '%s' is invalid: must be IPv4.", value(p.IP)))
		}
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateDelta(oc, current *OpenShiftCluster) error {
	err := immutable.ValidateWithPolicy("", oc, current, openShiftClusterUpdatePolicy)
	if err != nil {
		err := err.(*immutable.ValidationError)
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, err.Target, err.Message)
	}

	if current.UsesWorkloadIdentity() {
		for name := range current.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
			_, present := oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities[name]
			// this also validates that existing identities' names haven't changed
			if !present {
				return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, "properties.platformWorkloadIdentityProfile.platformWorkloadIdentities", "Operator identity cannot be removed or have its name changed.")
			}
		}
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validatePlatformWorkloadIdentityProfile(path string, pwip *generated.PlatformWorkloadIdentityProfile) error {
	// PlatformWorkloadIdentityProfile being empty is acceptable
	if pwip == nil {
		return nil
	}

	// Validate the PlatformWorkloadIdentities
	foundIdentityResourceIDs := map[string]string{}

	for name, p := range pwip.PlatformWorkloadIdentities {
		resourceID := value(p.ResourceID)
		if _, present := foundIdentityResourceIDs[strings.ToLower(resourceID)]; present {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, fmt.Sprintf("%s.PlatformWorkloadIdentities", path), fmt.Sprintf("ResourceID %s used by multiple identities.", strings.ToLower(resourceID)))
		}
		foundIdentityResourceIDs[strings.ToLower(resourceID)] = ""

		resource, err := azcorearm.ParseResourceID(resourceID)
		if err != nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, fmt.Sprintf("%s.PlatformWorkloadIdentities[%s].resourceID", path, name), fmt.Sprintf("ResourceID %s formatted incorrectly.", resourceID))
		}

		if name == "" {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, fmt.Sprintf("%s.PlatformWorkloadIdentities[%s].resourceID", path, name), "Operator name is empty.")
		}

		if resource.ResourceType.Type != "userAssignedIdentities" {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, fmt.Sprintf("%s.PlatformWorkloadIdentities[%s].resourceID", path, name), "Resource must be a user assigned identity.")
		}
	}

	if pwip.UpgradeableTo != nil {
		_, err := semver.NewVersion(*pwip.UpgradeableTo)
		if err != nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, fmt.Sprintf("%s.UpgradeableTo[%v]", path, *pwip.UpgradeableTo), "UpgradeableTo must be a valid OpenShift version in the format 'x.y.z'.")
		}
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validatePlatformIdentities(oc *OpenShiftCluster) error {
	pwip := oc.Properties.PlatformWorkloadIdentityProfile
	spp := oc.Properties.ServicePrincipalProfile

	if pwip == nil && spp == nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.servicePrincipalProfile", "Must provide either an identity or service principal credentials.")
	}

	if pwip != nil && spp != nil && (value(spp.ClientID) != "" || value(spp.ClientSecret) != "") {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.servicePrincipalProfile", "Cannot use identities and service principal credentials at the same time.")
	}

	clusterIdentityPresent := oc.Identity != nil
	operatorRolePresent := pwip != nil

	if clusterIdentityPresent != operatorRolePresent {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "identity", "Cluster identity and platform workload identities require each other.")
	}

	if clusterIdentityPresent && len(oc.Identity.UserAssignedIdentities) != 1 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "identity", "The provided cluster identity is invalid; there should be exactly one.")
	}

	if operatorRolePresent && len(pwip.PlatformWorkloadIdentities) == 0 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "properties.platformWorkloadIdentityProfile.platformWorkloadIdentities", "The set of platform workload identities cannot be empty.")
	}

	return nil
}
