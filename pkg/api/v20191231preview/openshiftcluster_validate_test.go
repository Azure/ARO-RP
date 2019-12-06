package v20191231preview

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"unicode"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/jim-minter/rp/pkg/api"
)

type validateTest struct {
	name    string
	modify  func(oc *OpenShiftCluster)
	wantErr string
}

var (
	subscriptionID = "af848f0a-dbe3-449f-9ccd-6f23ac6ef9f1"
	resourceGroup  = "resourcegroup"
	location       = "australiasoutheast"
	name           = "test-cluster"
	id             = fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/microsoft.redhatopenshift/openshiftclusters/%s", subscriptionID, resourceGroup, name)

	v = &validator{
		location:   location,
		resourceID: id,
		r: azure.Resource{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Provider:       "microsoft.redhatopenShift",
			ResourceType:   "openshiftclusters",
			ResourceName:   name,
		},
	}
)

func validOpenShiftCluster() *OpenShiftCluster {
	return &OpenShiftCluster{
		ID:       id,
		Name:     name,
		Type:     "Microsoft.RedHatOpenShift/openShiftClusters",
		Location: location,
		Tags:     Tags{"key": "value"},
		Properties: Properties{
			ProvisioningState: ProvisioningStateSucceeded,
			ServicePrincipalProfile: ServicePrincipalProfile{
				ClientID:     "2b5ba2c6-6205-4fc4-8b5d-9fea369ae1a2",
				ClientSecret: "secret",
			},
			NetworkProfile: NetworkProfile{
				PodCIDR:     "10.0.0.0/18",
				ServiceCIDR: "10.0.1.0/22",
			},
			MasterProfile: MasterProfile{
				VMSize:   VMSizeStandardD8sV3,
				SubnetID: fmt.Sprintf("/subscriptions/%s/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", subscriptionID),
			},
			WorkerProfiles: []WorkerProfile{
				{
					Name:       "worker",
					VMSize:     VMSizeStandardD4sV3,
					DiskSizeGB: 128,
					SubnetID:   fmt.Sprintf("/subscriptions/%s/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/worker", subscriptionID),
					Count:      3,
				},
			},
			APIServerURL: "url",
			ConsoleURL:   "url",
		},
	}
}

func runTests(t *testing.T, tests []*validateTest, f func(*OpenShiftCluster) error) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oc := validOpenShiftCluster()
			if tt.modify != nil {
				tt.modify(oc)
			}

			err := f(oc)
			if err == nil {
				if tt.wantErr != "" {
					t.Error(err)
				}

			} else {
				if err.Error() != tt.wantErr {
					t.Error(err)
				}

				validateCloudError(t, err)
			}
		})
	}
}

func validateCloudError(t *testing.T, err error) *api.CloudError {
	cloudErr, ok := err.(*api.CloudError)
	if !ok {
		t.Fatal("must return *api.CloudError")
	}

	if cloudErr.StatusCode != http.StatusBadRequest {
		t.Error(cloudErr.StatusCode)
	}
	if cloudErr.Code == "" {
		t.Error("code is required")
	}
	if cloudErr.Message == "" {
		t.Error("message is required")
	}
	if cloudErr.Target == "" {
		t.Error("target is required")
	}
	if cloudErr.Message != "" && !unicode.IsUpper(rune(cloudErr.Message[0])) {
		t.Error("message must start with upper case letter")
	}
	if strings.Contains(cloudErr.Message, `"`) {
		t.Error(`message must not contain '"'`)
	}
	if !strings.HasSuffix(cloudErr.Message, ".") {
		t.Error("message must end in '.'")
	}

	return cloudErr
}

func TestValidateOpenShiftCluster(t *testing.T) {
	tests := []*validateTest{
		{
			name: "valid",
		},
		{
			name: "id wrong",
			modify: func(oc *OpenShiftCluster) {
				oc.ID = "wrong"
			},
			wantErr: "400: MismatchingResourceID: id: The provided resource ID 'wrong' did not match the name in the Url '/subscriptions/af848f0a-dbe3-449f-9ccd-6f23ac6ef9f1/resourcegroups/resourcegroup/providers/microsoft.redhatopenshift/openshiftclusters/test-cluster'.",
		},
		{
			name: "name wrong",
			modify: func(oc *OpenShiftCluster) {
				oc.Name = "wrong"
			},
			wantErr: "400: MismatchingResourceName: name: The provided resource name 'wrong' did not match the name in the Url 'test-cluster'.",
		},
		{
			name: "type wrong",
			modify: func(oc *OpenShiftCluster) {
				oc.Type = "wrong"
			},
			wantErr: "400: MismatchingResourceType: type: The provided resource type 'wrong' did not match the name in the Url 'Microsoft.RedHatOpenShift/openShiftClusters'.",
		},
		{
			name: "location invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Location = "invalid"
			},
			wantErr: "400: InvalidParameter: location: The provided location 'invalid' is invalid.",
		},
	}

	runTests(t, tests, v.validateOpenShiftCluster)
}

func TestValidateProperties(t *testing.T) {
	tests := []*validateTest{
		{
			name: "valid",
		},
		{
			name: "provisioningState invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ProvisioningState = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.provisioningState: The provided provisioning state 'invalid' is invalid.",
		},
		{
			name: "no workerProfiles invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles = nil
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles: There should be exactly one worker profile.",
		},
		{
			name: "multiple workerProfiles invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles = []WorkerProfile{{}, {}}
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles: There should be exactly one worker profile.",
		},
		{
			name: "empty apiServerUrl valid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.APIServerURL = ""
			},
		},
		{
			name: "apiServerUrl invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.APIServerURL = "\x00"
			},
			wantErr: "400: InvalidParameter: properties.apiserverUrl: The provided API server URL '\x00' is invalid.",
		},
		{
			name: "empty consoleUrl valid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ConsoleURL = ""
			},
		},
		{
			name: "consoleUrl invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ConsoleURL = "\x00"
			},
			wantErr: "400: InvalidParameter: properties.consoleUrl: The provided console URL '\x00' is invalid.",
		},
	}

	runTests(t, tests, func(oc *OpenShiftCluster) error {
		return v.validateProperties("properties", &oc.Properties)
	})
}

func TestValidateServicePrincipalProfile(t *testing.T) {
	tests := []*validateTest{
		{
			name: "valid",
		},
		{
			name: "clientID invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ServicePrincipalProfile.ClientID = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.servicePrincipalProfile.clientId: The provided client ID 'invalid' is invalid.",
		},
		{
			name: "empty clientSecret invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ServicePrincipalProfile.ClientSecret = ""
			},
			wantErr: "400: InvalidParameter: properties.servicePrincipalProfile.clientSecret: The provided client secret is invalid.",
		},
	}

	runTests(t, tests, func(oc *OpenShiftCluster) error {
		return v.validateServicePrincipalProfile("properties.servicePrincipalProfile", &oc.Properties.ServicePrincipalProfile)
	})
}

func TestValidateNetworkProfile(t *testing.T) {
	tests := []*validateTest{
		{
			name: "valid",
		},
		{
			name: "podCidr invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.PodCIDR = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.networkProfile.podCidr: The provided pod CIDR 'invalid' is invalid: 'invalid CIDR address: invalid'.",
		},
		{
			name: "ipv6 podCidr invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.PodCIDR = "::0/0"
			},
			wantErr: "400: InvalidParameter: properties.networkProfile.podCidr: The provided pod CIDR '::0/0' is invalid: must be IPv4.",
		},
		{
			name: "serviceCidr invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.ServiceCIDR = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.networkProfile.serviceCidr: The provided service CIDR 'invalid' is invalid: 'invalid CIDR address: invalid'.",
		},
		{
			name: "ipv6 serviceCidr invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.ServiceCIDR = "::0/0"
			},
			wantErr: "400: InvalidParameter: properties.networkProfile.serviceCidr: The provided service CIDR '::0/0' is invalid: must be IPv4.",
		},
		{
			name: "podCidr too small",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.PodCIDR = "10.0.0.0/19"
			},
			wantErr: "400: InvalidParameter: properties.networkProfile.podCidr: The provided vnet CIDR '10.0.0.0/19' is invalid: must be /18 or larger.",
		},
		{
			name: "serviceCidr too small",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.ServiceCIDR = "10.0.0.0/23"
			},
			wantErr: "400: InvalidParameter: properties.networkProfile.serviceCidr: The provided vnet CIDR '10.0.0.0/23' is invalid: must be /22 or larger.",
		},
	}

	runTests(t, tests, func(oc *OpenShiftCluster) error {
		return v.validateNetworkProfile("properties.networkProfile", &oc.Properties.NetworkProfile)
	})
}

func TestValidateMasterProfile(t *testing.T) {
	tests := []*validateTest{
		{
			name: "valid",
		},
		{
			name: "vmSize unsupported",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.MasterProfile.VMSize = VMSizeStandardD2sV3
			},
			wantErr: "400: InvalidParameter: properties.masterProfile.vmSize: The provided master VM size 'Standard_D2s_v3' is invalid.",
		},
		{
			name: "subnetId invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.MasterProfile.SubnetID = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.masterProfile.subnetId: The provided master VM subnet 'invalid' is invalid.",
		},
		{
			name: "subnet subscriptionId not matching cluster subscriptionId",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.MasterProfile.SubnetID = "/subscriptions/7a3036d1-60a1-4605-8a41-44955e050804/resourcegroups/test-vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master"
			},
			wantErr: "400: InvalidParameter: properties.masterProfile.subnetId: The provided master VM subnet '/subscriptions/7a3036d1-60a1-4605-8a41-44955e050804/resourcegroups/test-vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master' is invalid: must be in same subscription as cluster.",
		},
	}

	runTests(t, tests, func(oc *OpenShiftCluster) error {
		return v.validateMasterProfile("properties.masterProfile", &oc.Properties.MasterProfile)
	})
}

func TestValidateWorkerProfile(t *testing.T) {
	tests := []*validateTest{
		{
			name: "valid",
		},
		{
			name: "name invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].Name = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles[0].name: The provided worker name 'invalid' is invalid.",
		},
		{
			name: "vmSize invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].VMSize = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles[0].vmSize: The provided worker VM size 'invalid' is invalid.",
		},
		{
			name: "disk too small",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].DiskSizeGB = 127
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles[0].diskSizeGB: The provided worker disk size '127' is invalid.",
		},
		{
			name: "subnetId invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].SubnetID = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles[0].subnetId: The provided worker VM subnet 'invalid' is invalid.",
		},
		{
			name: "master and worker subnets not in same vnet",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].SubnetID = fmt.Sprintf("/subscriptions/%s/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/different-vnet/subnets/worker", subscriptionID)
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles[0].subnetId: The provided worker VM subnet '/subscriptions/af848f0a-dbe3-449f-9ccd-6f23ac6ef9f1/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/different-vnet/subnets/worker' is invalid: must be in the same vnet as master VM subnet '/subscriptions/af848f0a-dbe3-449f-9ccd-6f23ac6ef9f1/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master'.",
		},
		{
			name: "master and worker subnets not different",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].SubnetID = oc.Properties.MasterProfile.SubnetID
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles[0].subnetId: The provided worker VM subnet '/subscriptions/af848f0a-dbe3-449f-9ccd-6f23ac6ef9f1/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master' is invalid: must be different to master VM subnet '/subscriptions/af848f0a-dbe3-449f-9ccd-6f23ac6ef9f1/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master'.",
		},
		{
			name: "count too small",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].Count = 2
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles[0].count: The provided worker count '2' is invalid.",
		},
		{
			name: "count too big",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].Count = 21
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles[0].count: The provided worker count '21' is invalid.",
		},
	}

	runTests(t, tests, func(oc *OpenShiftCluster) error {
		return v.validateWorkerProfile("properties.workerProfiles[0]", &oc.Properties.WorkerProfiles[0], &oc.Properties.MasterProfile)
	})
}

func TestOpenShiftClusterValidateDelta(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(oc *OpenShiftCluster)
		wantErr string
	}{
		{
			name:   "no change",
			modify: func(change *OpenShiftCluster) {},
		},
		{
			name:    "ID change",
			modify:  func(change *OpenShiftCluster) { change.ID = "foo" },
			wantErr: "400: PropertyChangeNotAllowed: id: Changing property 'id' is not allowed.",
		},
		{
			name:    "name change",
			modify:  func(change *OpenShiftCluster) { change.Name = "foo" },
			wantErr: "400: PropertyChangeNotAllowed: name: Changing property 'name' is not allowed.",
		},
		{
			name:    "type change",
			modify:  func(change *OpenShiftCluster) { change.Type = "foo" },
			wantErr: "400: PropertyChangeNotAllowed: type: Changing property 'type' is not allowed.",
		},
		{
			name:    "location change",
			modify:  func(change *OpenShiftCluster) { change.Location = "westus" },
			wantErr: "400: PropertyChangeNotAllowed: location: Changing property 'location' is not allowed.",
		},
		{
			name:   "tags change",
			modify: func(change *OpenShiftCluster) { change.Tags = Tags{"new": "value"} },
		},
		{
			name:    "provisioningstate change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ProvisioningState = ProvisioningStateUpdating },
			wantErr: "400: PropertyChangeNotAllowed: properties.provisioningState: Changing property 'properties.provisioningState' is not allowed.",
		},
		{
			name:    "APIServerURL change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.APIServerURL = "http://example.com" },
			wantErr: "400: PropertyChangeNotAllowed: properties.apiserverUrl: Changing property 'properties.apiserverUrl' is not allowed.",
		},
		{
			name:    "ConsoleURL change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ConsoleURL = "http://example.com" },
			wantErr: "400: PropertyChangeNotAllowed: properties.consoleUrl: Changing property 'properties.consoleUrl' is not allowed.",
		},
		{
			name:    "vmsize changed",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.MasterProfile.VMSize = VMSizeStandardD4sV3 },
			wantErr: "400: PropertyChangeNotAllowed: properties.masterProfile.vmSize: Changing property 'properties.masterProfile.vmSize' is not allowed.",
		},
		{
			name: "subnet changed",
			modify: func(oc *OpenShiftCluster) {
				subID := fmt.Sprintf("/subscriptions/%s/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/other", subscriptionID)
				oc.Properties.MasterProfile.SubnetID = subID
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.masterProfile.subnetId: Changing property 'properties.masterProfile.subnetId' is not allowed.",
		},
		{
			name:    "passwd change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ServicePrincipalProfile.ClientSecret = "frog" },
			wantErr: "400: PropertyChangeNotAllowed: properties.servicePrincipalProfile.clientSecret: Changing property 'properties.servicePrincipalProfile.clientSecret' is not allowed.",
		},
		{
			name:    "clientID change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ServicePrincipalProfile.ClientID = "fred" },
			wantErr: "400: PropertyChangeNotAllowed: properties.servicePrincipalProfile.clientId: Changing property 'properties.servicePrincipalProfile.clientId' is not allowed.",
		},
		{
			name:    "serviceCIDR change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.NetworkProfile.ServiceCIDR = "10.0.4.0/22" },
			wantErr: "400: PropertyChangeNotAllowed: properties.networkProfile.serviceCidr: Changing property 'properties.networkProfile.serviceCidr' is not allowed.",
		},
		{
			name:    "podCIDR change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.NetworkProfile.PodCIDR = "10.0.4.0/18" },
			wantErr: "400: PropertyChangeNotAllowed: properties.networkProfile.podCidr: Changing property 'properties.networkProfile.podCidr' is not allowed.",
		},
		{
			name: "name change",
			modify: func(oc *OpenShiftCluster) {
				wp := oc.Properties.WorkerProfiles[0]
				wp.Name = "notthis"
				oc.Properties.WorkerProfiles = []WorkerProfile{wp}
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles[0].name: Changing property 'properties.workerProfiles[0].name' is not allowed.",
		},
		{
			name: "vmsize change",
			modify: func(oc *OpenShiftCluster) {
				wp := oc.Properties.WorkerProfiles[0]
				wp.VMSize = VMSizeStandardD8sV3
				oc.Properties.WorkerProfiles = []WorkerProfile{wp}
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles[0].vmSize: Changing property 'properties.workerProfiles[0].vmSize' is not allowed.",
		},
		{
			name: "disksize change",
			modify: func(oc *OpenShiftCluster) {
				wp := oc.Properties.WorkerProfiles[0]
				wp.DiskSizeGB = 200
				oc.Properties.WorkerProfiles = []WorkerProfile{wp}
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles[0].diskSizeGB: Changing property 'properties.workerProfiles[0].diskSizeGB' is not allowed.",
		},
		{
			name: "subnet change",
			modify: func(oc *OpenShiftCluster) {
				subID := fmt.Sprintf("/subscriptions/%s/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/other", subscriptionID)
				wp := oc.Properties.WorkerProfiles[0]
				wp.SubnetID = subID
				oc.Properties.WorkerProfiles = []WorkerProfile{wp}
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles[0].subnetId: Changing property 'properties.workerProfiles[0].subnetId' is not allowed.",
		},
		{
			name: "new worker profile",
			modify: func(oc *OpenShiftCluster) {
				wp := oc.Properties.WorkerProfiles[0]
				oc.Properties.WorkerProfiles = []WorkerProfile{wp, wp}
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles: Changing property 'properties.workerProfiles' is not allowed.",
		},
		{
			name: "count can change",
			modify: func(oc *OpenShiftCluster) {
				wp := oc.Properties.WorkerProfiles[0]
				wp.Count = 15
				oc.Properties.WorkerProfiles = []WorkerProfile{wp}
			},
		},
	}
	old := validOpenShiftCluster()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oc := *old
			tt.modify(&oc)
			err := v.validateOpenShiftClusterDelta(&oc, old)
			if err == nil {
				if tt.wantErr != "" {
					t.Error(err)
				}
			} else {
				if err.Error() != tt.wantErr {
					t.Error(err)
				}

				validateCloudError(t, err)
			}

		})
	}
}
