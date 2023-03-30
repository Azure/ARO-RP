package v20230401

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
	"github.com/Azure/ARO-RP/pkg/api/validate"
	"github.com/Azure/ARO-RP/pkg/util/immutable"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

type openShiftClusterStaticValidator struct {
	location            string
	domain              string
	requireD2sV3Workers bool
	resourceID          string

	r azure.Resource
}

// Validate validates an OpenShift cluster
func (sv openShiftClusterStaticValidator) Static(_oc interface{}, _current *api.OpenShiftCluster, location, domain string, requireD2sV3Workers bool, resourceID string) error {
	sv.location = location
	sv.domain = domain
	sv.requireD2sV3Workers = requireD2sV3Workers
	sv.resourceID = resourceID

	oc := _oc.(*OpenShiftCluster)

	var current *OpenShiftCluster
	if _current != nil {
		current = (&openShiftClusterConverter{}).ToExternal(_current).(*OpenShiftCluster)
	}

	var err error
	sv.r, err = azure.ParseResourceID(sv.resourceID)
	if err != nil {
		return err
	}

	err = sv.validate(oc, current == nil)
	if err != nil {
		return err
	}

	if current == nil {
		return nil
	}

	return sv.validateDelta(oc, current)
}

func (sv openShiftClusterStaticValidator) validate(oc *OpenShiftCluster, isCreate bool) error {
	if !strings.EqualFold(oc.ID, sv.resourceID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeMismatchingResourceID, "id", "The provided resource ID '%s' did not match the name in the Url '%s'.", oc.ID, sv.resourceID)
	}
	if !strings.EqualFold(oc.Name, sv.r.ResourceName) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeMismatchingResourceName, "name", "The provided resource name '%s' did not match the name in the Url '%s'.", oc.Name, sv.r.ResourceName)
	}
	if !strings.EqualFold(oc.Type, resourceProviderNamespace+"/"+resourceType) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeMismatchingResourceType, "type", "The provided resource type '%s' did not match the name in the Url '%s'.", oc.Type, resourceProviderNamespace+"/"+resourceType)
	}
	if !strings.EqualFold(oc.Location, sv.location) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "location", "The provided location '%s' is invalid.", oc.Location)
	}

	return sv.validateProperties("properties", &oc.Properties, isCreate)
}

func (sv openShiftClusterStaticValidator) validateProperties(path string, p *OpenShiftClusterProperties, isCreate bool) error {
	switch p.ProvisioningState {
	case ProvisioningStateCreating, ProvisioningStateUpdating,
		ProvisioningStateAdminUpdating, ProvisioningStateDeleting,
		ProvisioningStateSucceeded, ProvisioningStateFailed:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".provisioningState", "The provided provisioning state '%s' is invalid.", p.ProvisioningState)
	}
	if err := sv.validateClusterProfile(path+".clusterProfile", &p.ClusterProfile, isCreate); err != nil {
		return err
	}
	if err := sv.validateConsoleProfile(path+".consoleProfile", &p.ConsoleProfile); err != nil {
		return err
	}
	if err := sv.validateServicePrincipalProfile(path+".servicePrincipalProfile", &p.ServicePrincipalProfile); err != nil {
		return err
	}
	if err := sv.validateNetworkProfile(path+".networkProfile", &p.NetworkProfile); err != nil {
		return err
	}
	if err := sv.validateMasterProfile(path+".masterProfile", &p.MasterProfile); err != nil {
		return err
	}
	if err := sv.validateAPIServerProfile(path+".apiserverProfile", &p.APIServerProfile); err != nil {
		return err
	}

	if isCreate {
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
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".domain", "The provided domain '%s' is invalid.", cp.Domain)
		}
	} else {
		// We currently do not allow domains with a digit as a first charecter,
		// for new clusters, but we already have some existing clusters with
		// domains like this and we need to allow customers to update them.
		if !validate.RxDomainNameRFC1123.MatchString(cp.Domain) {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".domain", "The provided domain '%s' is invalid.", cp.Domain)
		}
	}
	// domain ends .aroapp.io, but doesn't end .<rp-location>.aroapp.io
	if strings.HasSuffix(cp.Domain, "."+strings.SplitN(sv.domain, ".", 2)[1]) &&
		!strings.HasSuffix(cp.Domain, "."+sv.domain) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".domain", "The provided domain '%s' is invalid.", cp.Domain)
	}
	// domain is of form multiple.names.<rp-location>.aroapp.io
	if strings.HasSuffix(cp.Domain, "."+sv.domain) &&
		strings.ContainsRune(strings.TrimSuffix(cp.Domain, "."+sv.domain), '.') {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".domain", "The provided domain '%s' is invalid.", cp.Domain)
	}

	if !validate.RxResourceGroupID.MatchString(cp.ResourceGroupID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".resourceGroupId", "The provided resource group '%s' is invalid.", cp.ResourceGroupID)
	}
	if strings.Split(cp.ResourceGroupID, "/")[2] != sv.r.SubscriptionID {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".resourceGroupId", "The provided resource group '%s' is invalid: must be in same subscription as cluster.", cp.ResourceGroupID)
	}
	if strings.EqualFold(cp.ResourceGroupID, fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", sv.r.SubscriptionID, sv.r.ResourceGroup)) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".resourceGroupId", "The provided resource group '%s' is invalid: must be different from resourceGroup of the OpenShift cluster object.", cp.ResourceGroupID)
	}

	switch cp.FipsValidatedModules {
	case FipsValidatedModulesDisabled, FipsValidatedModulesEnabled:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".fipsValidatedModules", "The provided value '%s' is invalid.", cp.FipsValidatedModules)
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateConsoleProfile(path string, cp *ConsoleProfile) error {
	if cp.URL != "" {
		if _, err := url.Parse(cp.URL); err != nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".url", "The provided console URL '%s' is invalid.", cp.URL)
		}
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateServicePrincipalProfile(path string, spp *ServicePrincipalProfile) error {
	valid := uuid.IsValid(spp.ClientID)
	if !valid {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".clientId", "The provided client ID '%s' is invalid.", spp.ClientID)
	}
	if spp.ClientSecret == "" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".clientSecret", "The provided client secret is invalid.")
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateNetworkProfile(path string, np *NetworkProfile) error {
	_, pod, err := net.ParseCIDR(np.PodCIDR)
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".podCidr", "The provided pod CIDR '%s' is invalid: '%s'.", np.PodCIDR, err)
	}
	if pod.IP.To4() == nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".podCidr", "The provided pod CIDR '%s' is invalid: must be IPv4.", np.PodCIDR)
	}
	{
		ones, _ := pod.Mask.Size()
		if ones > 18 {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".podCidr", "The provided vnet CIDR '%s' is invalid: must be /18 or larger.", np.PodCIDR)
		}
	}
	_, service, err := net.ParseCIDR(np.ServiceCIDR)
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".serviceCidr", "The provided service CIDR '%s' is invalid: '%s'.", np.ServiceCIDR, err)
	}
	if service.IP.To4() == nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".serviceCidr", "The provided service CIDR '%s' is invalid: must be IPv4.", np.ServiceCIDR)
	}
	{
		ones, _ := service.Mask.Size()
		if ones > 22 {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".serviceCidr", "The provided vnet CIDR '%s' is invalid: must be /22 or larger.", np.ServiceCIDR)
		}
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateMasterProfile(path string, mp *MasterProfile) error {
	if !validate.VMSizeIsValid(api.VMSize(mp.VMSize), sv.requireD2sV3Workers, true) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".vmSize", "The provided master VM size '%s' is invalid.", mp.VMSize)
	}
	if !validate.RxSubnetID.MatchString(mp.SubnetID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", "The provided master VM subnet '%s' is invalid.", mp.SubnetID)
	}
	sr, err := azure.ParseResourceID(mp.SubnetID)
	if err != nil {
		return err
	}
	if sr.SubscriptionID != sv.r.SubscriptionID {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", "The provided master VM subnet '%s' is invalid: must be in same subscription as cluster.", mp.SubnetID)
	}
	switch mp.EncryptionAtHost {
	case EncryptionAtHostDisabled, EncryptionAtHostEnabled:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".encryptionAtHost", "The provided value '%s' is invalid.", mp.EncryptionAtHost)
	}
	if mp.DiskEncryptionSetID != "" {
		if !validate.RxDiskEncryptionSetID.MatchString(mp.DiskEncryptionSetID) {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".diskEncryptionSetId", "The provided master disk encryption set '%s' is invalid.", mp.DiskEncryptionSetID)
		}
		desr, err := azure.ParseResourceID(mp.DiskEncryptionSetID)
		if err != nil {
			return err
		}
		if desr.SubscriptionID != sv.r.SubscriptionID {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".diskEncryptionSetId", "The provided master disk encryption set '%s' is invalid: must be in same subscription as cluster.", mp.DiskEncryptionSetID)
		}
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateWorkerProfile(path string, wp *WorkerProfile, mp *MasterProfile) error {
	if wp.Name != "worker" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".name", "The provided worker name '%s' is invalid.", wp.Name)
	}
	if !validate.VMSizeIsValid(api.VMSize(wp.VMSize), sv.requireD2sV3Workers, false) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".vmSize", "The provided worker VM size '%s' is invalid.", wp.VMSize)
	}
	if !validate.DiskSizeIsValid(wp.DiskSizeGB) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".diskSizeGB", "The provided worker disk size '%d' is invalid.", wp.DiskSizeGB)
	}
	if !validate.RxSubnetID.MatchString(wp.SubnetID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", "The provided worker VM subnet '%s' is invalid.", wp.SubnetID)
	}
	switch wp.EncryptionAtHost {
	case EncryptionAtHostDisabled, EncryptionAtHostEnabled:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".encryptionAtHost", "The provided value '%s' is invalid.", wp.EncryptionAtHost)
	}
	workerVnetID, _, err := subnet.Split(wp.SubnetID)
	if err != nil {
		return err
	}
	masterVnetID, _, err := subnet.Split(mp.SubnetID)
	if err != nil {
		return err
	}
	if !strings.EqualFold(masterVnetID, workerVnetID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", "The provided worker VM subnet '%s' is invalid: must be in the same vnet as master VM subnet '%s'.", wp.SubnetID, mp.SubnetID)
	}
	if strings.EqualFold(mp.SubnetID, wp.SubnetID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", "The provided worker VM subnet '%s' is invalid: must be different to master VM subnet '%s'.", wp.SubnetID, mp.SubnetID)
	}
	if wp.Count < 2 || wp.Count > 50 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".count", "The provided worker count '%d' is invalid.", wp.Count)
	}
	if !strings.EqualFold(mp.DiskEncryptionSetID, wp.DiskEncryptionSetID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", "The provided worker disk encryption set '%s' is invalid: must be the same as master disk encryption set '%s'.", wp.DiskEncryptionSetID, mp.DiskEncryptionSetID)
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateAPIServerProfile(path string, ap *APIServerProfile) error {
	switch ap.Visibility {
	case VisibilityPublic, VisibilityPrivate:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".visibility", "The provided visibility '%s' is invalid.", ap.Visibility)
	}
	if ap.URL != "" {
		if _, err := url.Parse(ap.URL); err != nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".url", "The provided URL '%s' is invalid.", ap.URL)
		}
	}
	if ap.IP != "" {
		ip := net.ParseIP(ap.IP)
		if ip == nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".ip", "The provided IP '%s' is invalid.", ap.IP)
		}
		if ip.To4() == nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".ip", "The provided IP '%s' is invalid: must be IPv4.", ap.IP)
		}
	}

	return nil
}

func (sv openShiftClusterStaticValidator) validateIngressProfile(path string, p *IngressProfile) error {
	if p.Name != "default" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".name", "The provided ingress name '%s' is invalid.", p.Name)
	}
	switch p.Visibility {
	case VisibilityPublic, VisibilityPrivate:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".visibility", "The provided visibility '%s' is invalid.", p.Visibility)
	}
	if p.IP != "" {
		ip := net.ParseIP(p.IP)
		if ip == nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".ip", "The provided IP '%s' is invalid.", p.IP)
		}
		if ip.To4() == nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".ip", "The provided IP '%s' is invalid: must be IPv4.", p.IP)
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
