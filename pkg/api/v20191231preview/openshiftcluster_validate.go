package v20191231preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	uuid "github.com/satori/go.uuid"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/immutable"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

var (
	rxSubnetID   = regexp.MustCompile(`(?i)^/subscriptions/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/resourceGroups/[-a-z0-9_().]{0,89}[-a-z0-9_()]/providers/Microsoft\.Network/virtualNetworks/[-a-z0-9_.]{2,64}/subnets/[-a-z0-9_.]{2,80}$`)
	rxDomainName = regexp.MustCompile(`^` +
		`([a-z0-9]|[a-z0-9][-a-z0-9]{0,61}[a-z0-9])` +
		`(\.([a-z0-9]|[a-z0-9][-a-z0-9]{0,61}[a-z0-9]))*` +
		`$`)
)

type openShiftClusterStaticValidator struct {
	location   string
	resourceID string
	r          azure.Resource
}

// validateOpenShiftCluster validates an OpenShift cluster
func validateOpenShiftCluster(location, resourceID string, oc, current *OpenShiftCluster) error {
	r, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return err
	}

	sv := &openShiftClusterStaticValidator{
		location:   location,
		resourceID: resourceID,
		r:          r,
	}

	err = sv.validate(oc)
	if err != nil {
		return err
	}

	if current == nil {
		return nil
	}

	return sv.validateDelta(oc, current)
}

func (sv *openShiftClusterStaticValidator) validate(oc *OpenShiftCluster) error {
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

	return sv.validateProperties("properties", &oc.Properties)
}

func (sv *openShiftClusterStaticValidator) validateProperties(path string, p *Properties) error {
	switch p.ProvisioningState {
	case ProvisioningStateCreating, ProvisioningStateUpdating,
		ProvisioningStateDeleting, ProvisioningStateSucceeded,
		ProvisioningStateFailed:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".provisioningState", "The provided provisioning state '%s' is invalid.", p.ProvisioningState)
	}
	if !rxDomainName.MatchString(p.ClusterDomain) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".clusterDomain", "The provided cluster domain '%s' is invalid.", p.ClusterDomain)
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
	if len(p.WorkerProfiles) != 1 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".workerProfiles", "There should be exactly one worker profile.")
	}
	if err := sv.validateWorkerProfile(path+".workerProfiles['"+p.WorkerProfiles[0].Name+"']", &p.WorkerProfiles[0], &p.MasterProfile); err != nil {
		return err
	}
	if err := sv.validateAPIServerProfile(path+".apiserverProfile", &p.APIServerProfile); err != nil {
		return err
	}
	if len(p.IngressProfiles) != 1 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".ingressProfiles", "There should be exactly one ingress profile.")
	}
	if err := sv.validateIngressProfile(path+".ingressProfiles['"+p.IngressProfiles[0].Name+"']", &p.IngressProfiles[0]); err != nil {
		return err
	}
	if p.ConsoleURL != "" {
		if _, err := url.Parse(p.ConsoleURL); err != nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".consoleUrl", "The provided console URL '%s' is invalid.", p.ConsoleURL)
		}
	}

	return nil
}

func (sv *openShiftClusterStaticValidator) validateServicePrincipalProfile(path string, spp *ServicePrincipalProfile) error {
	_, err := uuid.FromString(spp.ClientID)
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".clientId", "The provided client ID '%s' is invalid.", spp.ClientID)
	}
	if spp.ClientSecret == "" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".clientSecret", "The provided client secret is invalid.")
	}

	return nil
}

func (sv *openShiftClusterStaticValidator) validateNetworkProfile(path string, np *NetworkProfile) error {
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

func (sv *openShiftClusterStaticValidator) validateMasterProfile(path string, mp *MasterProfile) error {
	switch mp.VMSize {
	case VMSizeStandardD8sV3:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".vmSize", "The provided master VM size '%s' is invalid.", mp.VMSize)
	}
	if !rxSubnetID.MatchString(mp.SubnetID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", "The provided master VM subnet '%s' is invalid.", mp.SubnetID)
	}
	sr, err := azure.ParseResourceID(mp.SubnetID)
	if err != nil {
		return err
	}
	if sr.SubscriptionID != sv.r.SubscriptionID {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", "The provided master VM subnet '%s' is invalid: must be in same subscription as cluster.", mp.SubnetID)
	}

	return nil
}

func (sv *openShiftClusterStaticValidator) validateWorkerProfile(path string, wp *WorkerProfile, mp *MasterProfile) error {
	if wp.Name != "worker" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".name", "The provided worker name '%s' is invalid.", wp.Name)
	}
	switch wp.VMSize {
	case VMSizeStandardD2sV3, VMSizeStandardD4sV3, VMSizeStandardD8sV3:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".vmSize", "The provided worker VM size '%s' is invalid.", wp.VMSize)
	}
	if wp.DiskSizeGB < 128 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".diskSizeGB", "The provided worker disk size '%d' is invalid.", wp.DiskSizeGB)
	}
	if !rxSubnetID.MatchString(wp.SubnetID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", "The provided worker VM subnet '%s' is invalid.", wp.SubnetID)
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
	if wp.Count < 3 || wp.Count > 20 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".count", "The provided worker count '%d' is invalid.", wp.Count)
	}

	return nil
}

func (sv *openShiftClusterStaticValidator) validateAPIServerProfile(path string, ap *APIServerProfile) error {
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

func (sv *openShiftClusterStaticValidator) validateIngressProfile(path string, p *IngressProfile) error {
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

func (sv *openShiftClusterStaticValidator) validateDelta(oc, current *OpenShiftCluster) error {
	return immutable.Validate("", oc, current)
}
