package v20191231preview

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/apparentlymart/go-cidr/cidr"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/util/resource"
)

// Validate validates an OpenShift cluster
func (oc *OpenShiftCluster) Validate(resourceID string, current *api.OpenShiftCluster) error {
	err := oc.validate(resourceID)
	if err != nil {
		return err
	}

	if current == nil {
		return nil
	}

	return oc.validateDelta(OpenShiftClusterToExternal(current))
}

func (oc *OpenShiftCluster) validate(resourceID string) error {
	resourceName, err := resource.ResourceName(resourceID)
	if err != nil {
		return err
	}

	if !strings.EqualFold(oc.ID, resourceID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeMismatchingResourceID, "id", "The provided resource ID '%s' did not match the name in the Url '%s'.", oc.ID, resourceID)
	}
	if !strings.EqualFold(oc.Name, resourceName) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeMismatchingResourceName, "name", "The provided resource name '%s' did not match the name in the Url '%s'.", oc.Name, resourceName)
	}
	if !strings.EqualFold(oc.Type, resourceProviderNamespace+"/"+resourceType) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeMismatchingResourceType, "type", "The provided resource type '%s' did not match the name in the Url '%s'.", oc.Type, resourceProviderNamespace+"/"+resourceType)
	}
	if !strings.EqualFold(oc.Location, os.Getenv("LOCATION")) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "location", "The provided location '%s' is invalid.", oc.Location)
	}

	return oc.Properties.validate("properties")
}

func (p *Properties) validate(path string) error {
	switch p.ProvisioningState {
	case ProvisioningStateUpdating, ProvisioningStateDeleting,
		ProvisioningStateSucceeded, ProvisioningStateFailed:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".provisioningState", "The provided provisioning state '%s' is invalid.", p.ProvisioningState)
	}
	if err := p.NetworkProfile.validate(path + ".networkProfile"); err != nil {
		return err
	}
	if err := p.MasterProfile.validate(path + ".masterProfile"); err != nil {
		return err
	}
	if len(p.WorkerProfiles) != 1 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".workerProfiles", "There should be exactly one worker profile.")
	}
	if err := p.WorkerProfiles[0].validate(path + `.workerProfiles["` + p.WorkerProfiles[0].Name + `"]`); err != nil {
		return err
	}
	if p.APIServerURL != "" {
		if _, err := url.Parse(p.APIServerURL); err != nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".apiserverURL", "The provided API server URL '%s' is invalid.", p.APIServerURL)
		}
	}
	if p.ConsoleURL != "" {
		if _, err := url.Parse(p.ConsoleURL); err != nil {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".consoleURL", "The provided console URL '%s' is invalid.", p.ConsoleURL)
		}
	}

	return nil
}

func (np *NetworkProfile) validate(path string) error {
	_, vnet, err := net.ParseCIDR(np.VNetCIDR)
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".vnetCidr", "The provided vnet CIDR '%s' is invalid: %q.", np.VNetCIDR, err)
	}
	if vnet.IP.To4() == nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".vnetCidr", "The provided vnet CIDR '%s' is invalid: must be IPv4.", np.VNetCIDR)
	}
	{
		ones, _ := vnet.Mask.Size()
		if ones > 24 {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".vnetCidr", "The provided vnet CIDR '%s' is invalid: must be /24 or larger.", np.VNetCIDR)
		}
	}
	_, pod, err := net.ParseCIDR(np.PodCIDR)
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".podCidr", "The provided pod CIDR '%s' is invalid: %q.", np.PodCIDR, err)
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
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".serviceCidr", "The provided service CIDR '%s' is invalid: %q.", np.ServiceCIDR, err)
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

	if err = cidr.VerifyNoOverlap([]*net.IPNet{vnet, pod, service}, &net.IPNet{IP: net.IPv4zero, Mask: net.IPMask(net.IPv4zero)}); err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "", "The provided CIDRs must not overlap: %q.", err)
	}

	return nil
}

func (mp *MasterProfile) validate(path string) error {
	switch mp.VMSize {
	case VMSizeStandardD8sV3:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".vmSize", "The provided VM size '%s' is invalid.", mp.VMSize)
	}
	return nil
}

func (wp *WorkerProfile) validate(path string) error {
	if wp.Name != "worker" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".name", "The provided name '%s' is invalid.", wp.Name)
	}
	switch wp.VMSize {
	case VMSizeStandardD2sV3, VMSizeStandardD4sV3, VMSizeStandardD8sV3:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".vmSize", "The provided VM size '%s' is invalid.", wp.VMSize)
	}
	if wp.DiskSizeGB < 128 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".diskSizeGB", "The provided disk size '%d' is invalid.", wp.DiskSizeGB)
	}
	if wp.Count < 3 || wp.Count > 20 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".count", "The provided count '%d' is invalid.", wp.Count)
	}

	return nil
}

func (oc *OpenShiftCluster) validateDelta(current *OpenShiftCluster) error {
	if !strings.EqualFold(current.ID, oc.ID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, "id", "Changing property 'id' is not allowed.")
	}
	if !strings.EqualFold(current.Name, oc.Name) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, "name", "Changing property 'name' is not allowed.")
	}
	if !strings.EqualFold(current.Type, oc.Type) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, "type", "Changing property 'type' is not allowed.")
	}
	if !strings.EqualFold(current.Location, oc.Location) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, "location", "Changing property 'location' is not allowed.")
	}

	return oc.Properties.validateDelta("properties", &current.Properties)
}

func (p *Properties) validateDelta(path string, current *Properties) error {
	if current.ProvisioningState != p.ProvisioningState {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, path+".provisioningState", "Changing property '"+path+".provisioningState' is not allowed.")
	}
	if err := p.NetworkProfile.validateDelta(path+".networkProfile", &current.NetworkProfile); err != nil {
		return err
	}
	if err := p.MasterProfile.validateDelta(path+".masterProfile", &current.MasterProfile); err != nil {
		return err
	}
	if err := p.WorkerProfiles[0].validateDelta(path+`.workerProfiles["`+p.WorkerProfiles[0].Name+`"]`, &current.WorkerProfiles[0]); err != nil {
		return err
	}
	if current.APIServerURL != p.APIServerURL {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, path+".apiserverURL", "Changing property '"+path+".apiserverURL' is not allowed.")
	}
	if current.ConsoleURL != p.ConsoleURL {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, path+".consoleURL", "Changing property '"+path+".consoleURL' is not allowed.")
	}

	return nil
}

func (np *NetworkProfile) validateDelta(path string, current *NetworkProfile) error {
	if current.VNetCIDR != np.VNetCIDR {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, path+".vnetCidr", "Changing property '"+path+".vnetCidr' is not allowed.")
	}
	if current.PodCIDR != np.PodCIDR {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, path+".podCidr", "Changing property '"+path+".podCidr' is not allowed.")
	}
	if current.ServiceCIDR != np.ServiceCIDR {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, path+".serviceCidr", "Changing property '"+path+".serviceCidr' is not allowed.")
	}

	return nil
}

func (mp *MasterProfile) validateDelta(path string, current *MasterProfile) error {
	if current.VMSize != mp.VMSize {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, path+".vmSize", "Changing property '"+path+".vmSize' is not allowed.")
	}

	return nil
}

func (wp *WorkerProfile) validateDelta(path string, current *WorkerProfile) error {
	if current.Name != wp.Name {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, path+".name", "Changing property '"+path+".name' is not allowed.")
	}
	if current.VMSize != wp.VMSize {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, path+".vmSize", "Changing property '"+path+".vmSize' is not allowed.")
	}
	if current.DiskSizeGB != wp.DiskSizeGB {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, path+".diskSizeGB", "Changing property '"+path+".diskSizeGB' is not allowed.")
	}

	return nil
}
