package openstack

import (
	"github.com/openshift/installer/pkg/types/openstack/validation"
)

type realValidValuesFetcher struct{}

// NewValidValuesFetcher returns a new ValidValuesFetcher.
func NewValidValuesFetcher() validation.ValidValuesFetcher {
	return realValidValuesFetcher{}
}

// GetCloudNames gets the valid cloud names. These are read from clouds.yaml.
func (f realValidValuesFetcher) GetCloudNames() ([]string, error) {
	cloudNames := make([]string, 0)
	return cloudNames, nil
}

// GetNetworkNames gets the valid network names.
func (f realValidValuesFetcher) GetNetworkNames(cloud string) ([]string, error) {
	networkNames := make([]string, 0)
	return networkNames, nil
}

// GetFlavorNames gets a list of valid flavor names.
func (f realValidValuesFetcher) GetFlavorNames(cloud string) ([]string, error) {
	flavorNames := make([]string, 0)
	return flavorNames, nil
}

func (f realValidValuesFetcher) GetNetworkExtensionsAliases(cloud string) ([]string, error) {
	extAliases := make([]string, 0)
	return extAliases, nil
}

func (f realValidValuesFetcher) GetServiceCatalog(cloud string) ([]string, error) {
	serviceCatalogNames := make([]string, 0)
	return serviceCatalogNames, nil
}

func (f realValidValuesFetcher) GetFloatingIPNames(cloud string, floatingNetworkName string) ([]string, error) {
	floatingIPNames := make([]string, 0)
	return floatingIPNames, nil
}

func (f realValidValuesFetcher) GetSubnetCIDR(cloud string, subnetID string) (string, error) {
	return "", nil
}
