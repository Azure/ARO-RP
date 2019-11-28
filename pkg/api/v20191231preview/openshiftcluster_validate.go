package v20191231preview

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/apparentlymart/go-cidr/cidr"
	uuid "github.com/satori/go.uuid"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/util/subnet"
)

var (
	rxSubnetID = regexp.MustCompile(`(?i)^/subscriptions/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/resourceGroups/[-a-z0-9_().]{0,89}[-a-z0-9_()]/providers/Microsoft\.Network/virtualNetworks/[-a-z0-9_.]{2,64}/subnets/[-a-z0-9_.]{2,80}$`)
)

// Validate validates an OpenShift cluster
func (oc *OpenShiftCluster) Validate(ctx context.Context, resourceID string, current *api.OpenShiftCluster) error {
	err := oc.validate(resourceID)
	if err != nil {
		return err
	}

	internal := &api.OpenShiftCluster{}
	oc.ToInternal(internal)

	masterSubnet, err := subnet.Get(ctx, &internal.Properties.ServicePrincipalProfile, internal.Properties.MasterProfile.SubnetID)
	if err != nil {
		// TODO: return friendly error if SP is not authorised
		if err, ok := err.(autorest.DetailedError); ok && err.StatusCode == http.StatusNotFound {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "properties.masterProfile.subnetId", "The provided master VM subnet '%s' could not be found.", oc.Properties.MasterProfile.SubnetID)
		}
		return err
	}

	workerSubnet, err := subnet.Get(ctx, &internal.Properties.ServicePrincipalProfile, internal.Properties.WorkerProfiles[0].SubnetID)
	if err != nil {
		// TODO: return friendly error if SP is not authorised
		if err, ok := err.(autorest.DetailedError); ok && err.StatusCode == http.StatusNotFound {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "properties.masterProfile.subnetId", "The provided master VM subnet '%s' could not be found.", oc.Properties.MasterProfile.SubnetID)
		}
		return err
	}

	err = oc.validateSubnets(masterSubnet, workerSubnet)
	if err != nil {
		return err
	}

	if current == nil {
		return nil
	}

	return oc.validateDelta(OpenShiftClusterToExternal(current))
}

func (oc *OpenShiftCluster) validate(resourceID string) error {
	r, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return err
	}

	if !strings.EqualFold(oc.ID, resourceID) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeMismatchingResourceID, "id", "The provided resource ID '%s' did not match the name in the Url '%s'.", oc.ID, resourceID)
	}
	if !strings.EqualFold(oc.Name, r.ResourceName) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeMismatchingResourceName, "name", "The provided resource name '%s' did not match the name in the Url '%s'.", oc.Name, r.ResourceName)
	}
	if !strings.EqualFold(oc.Type, resourceProviderNamespace+"/"+resourceType) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeMismatchingResourceType, "type", "The provided resource type '%s' did not match the name in the Url '%s'.", oc.Type, resourceProviderNamespace+"/"+resourceType)
	}
	if !strings.EqualFold(oc.Location, os.Getenv("LOCATION")) {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "location", "The provided location '%s' is invalid.", oc.Location)
	}

	return oc.Properties.validate("properties", resourceID)
}

func (p *Properties) validate(path string, resourceID string) error {
	switch p.ProvisioningState {
	case ProvisioningStateCreating, ProvisioningStateUpdating,
		ProvisioningStateDeleting, ProvisioningStateSucceeded,
		ProvisioningStateFailed:
	default:
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".provisioningState", "The provided provisioning state '%s' is invalid.", p.ProvisioningState)
	}
	if err := p.ServicePrincipalProfile.validate(path + ".servicePrincipalProfile"); err != nil {
		return err
	}
	if err := p.NetworkProfile.validate(path + ".networkProfile"); err != nil {
		return err
	}
	if err := p.MasterProfile.validate(path+".masterProfile", resourceID); err != nil {
		return err
	}
	if len(p.WorkerProfiles) != 1 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".workerProfiles", "There should be exactly one worker profile.")
	}
	if err := p.WorkerProfiles[0].validate(path+`.workerProfiles["`+p.WorkerProfiles[0].Name+`"]`, &p.MasterProfile); err != nil {
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

func (spp *ServicePrincipalProfile) validate(path string) error {
	_, err := uuid.FromString(spp.ClientID)
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".clientId", "The provided client ID '%s' is invalid.", spp.ClientID)
	}
	if spp.ClientSecret == "" {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".cilentSecret", "The provided client secret is invalid.")
	}

	return nil
}

func (np *NetworkProfile) validate(path string) error {
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

	return nil
}

func (mp *MasterProfile) validate(path, resourceID string) error {
	r, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return err
	}

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
	if sr.SubscriptionID != r.SubscriptionID {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", "The provided master VM subnet '%s' is invalid: must be in same subscription as cluster.", mp.SubnetID)
	}

	return nil
}

func (wp *WorkerProfile) validate(path string, mp *MasterProfile) error {
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
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".subnetId", "The provided worker VM subnet '%s' is invalid: must be in the same vnet as master VM subnet '%s'", wp.SubnetID, mp.SubnetID)
	}
	if wp.Count < 3 || wp.Count > 20 {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, path+".count", "The provided worker count '%d' is invalid.", wp.Count)
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
	if err := p.ServicePrincipalProfile.validateDelta(path+".servicePrincipalProfile", &current.ServicePrincipalProfile); err != nil {
		return err
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

func (spp *ServicePrincipalProfile) validateDelta(path string, current *ServicePrincipalProfile) error {
	if current.ClientID != spp.ClientID {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, path+".vnetCidr", "Changing property '"+path+".clientId' is not allowed.")
	}
	if current.ClientSecret != spp.ClientSecret {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, path+".podCidr", "Changing property '"+path+".clientSecret' is not allowed.")
	}

	return nil
}

func (np *NetworkProfile) validateDelta(path string, current *NetworkProfile) error {
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
	if current.SubnetID != mp.SubnetID {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, path+".subnetId", "Changing property '"+path+".subnetId' is not allowed.")
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
	if current.SubnetID != wp.SubnetID {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodePropertyChangeNotAllowed, path+".subnetId", "Changing property '"+path+".subnetId' is not allowed.")
	}

	return nil
}

func (oc *OpenShiftCluster) validateSubnets(masterSubnet, workerSubnet *network.Subnet) error {
	if masterSubnet.SubnetPropertiesFormat != nil && masterSubnet.SubnetPropertiesFormat.NetworkSecurityGroup != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "properties.masterProfile.subnetId", "The provided master VM subnet '%s' is invalid: must not have a network security group attached.", oc.Properties.MasterProfile.SubnetID)
	}

	_, master, err := net.ParseCIDR(*masterSubnet.AddressPrefix)
	if err != nil {
		return err
	}
	{
		ones, _ := master.Mask.Size()
		if ones > 27 {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "properties.masterProfile.subnetId", "The provided master VM subnet '%s' is invalid: must be /27 or larger.", oc.Properties.MasterProfile.SubnetID)
		}
	}

	if workerSubnet.SubnetPropertiesFormat != nil && workerSubnet.SubnetPropertiesFormat.NetworkSecurityGroup != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, `properties.workerProfiles["worker"].subnetId`, "The provided worker VM subnet '%s' is invalid: must not have a network security group attached.", oc.Properties.WorkerProfiles[0].SubnetID)
	}

	_, worker, err := net.ParseCIDR(*workerSubnet.AddressPrefix)
	if err != nil {
		return err
	}
	{
		ones, _ := worker.Mask.Size()
		if ones > 27 {
			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, `properties.workerProfiles["worker"].subnetId`, "The provided worker VM subnet '%s' is invalid: must be /27 or larger.", oc.Properties.WorkerProfiles[0].SubnetID)
		}
	}

	_, pod, err := net.ParseCIDR(oc.Properties.NetworkProfile.PodCIDR)
	if err != nil {
		return err
	}

	_, service, err := net.ParseCIDR(oc.Properties.NetworkProfile.ServiceCIDR)
	if err != nil {
		return err
	}

	err = cidr.VerifyNoOverlap([]*net.IPNet{master, worker, pod, service}, &net.IPNet{IP: net.IPv4zero, Mask: net.IPMask(net.IPv4zero)})
	if err != nil {
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The provided CIDRs must not overlap: %q.", err)
	}

	return nil
}
