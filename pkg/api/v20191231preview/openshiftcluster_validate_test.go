package v20191231preview

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/azure"
	uuid "github.com/satori/go.uuid"

	"github.com/jim-minter/rp/pkg/api"
)

type validateTest struct {
	name    string
	f       func(change *OpenShiftCluster)
	wantErr error
}

var (
	subID        = uuid.NewV4().String()
	testLocation = "australiasoutheast"
	clusterName  = "test-cluster"
	resID        = fmt.Sprintf("/subscriptions/%s/resourcegroups/%s/providers/Microsoft.RedHatOpenShift/openShiftClusters/%s", subID, clusterName, clusterName)
	vnetID       = fmt.Sprintf("/subscriptions/%s/resourcegroups/test-vnet/providers/Microsoft.Network/virtualNetworks/test-vnet", subID)
	subnetID     = vnetID + "/subnets/master"
	goodOC       = OpenShiftCluster{
		ID:       resID,
		Name:     clusterName,
		Type:     "Microsoft.RedHatOpenShift/openShiftClusters",
		Location: testLocation,
		Tags:     map[string]string{"foo": "fee"}, // not validated
		Properties: Properties{
			ProvisioningState: ProvisioningStateSucceeded,
			APIServerURL:      "url",
			ConsoleURL:        "url",
			ServicePrincipalProfile: ServicePrincipalProfile{
				ClientID:     uuid.NewV4().String(),
				ClientSecret: "foo",
			},
			NetworkProfile: NetworkProfile{
				PodCIDR:     "10.0.0.0/18",
				ServiceCIDR: "10.0.1.0/22",
			},
			MasterProfile: MasterProfile{
				VMSize:   VMSizeStandardD8sV3,
				SubnetID: subnetID,
			},
			WorkerProfiles: []WorkerProfile{
				{
					Name:       "worker",
					VMSize:     VMSizeStandardD4sV3,
					DiskSizeGB: 128,
					SubnetID:   subnetID,
					Count:      12,
				},
			},
		},
	}
)

func TestOpenShiftClusterValidate(t *testing.T) {
	tests := []validateTest{
		{
			name: "pass",
			f:    func(change *OpenShiftCluster) {},
		},
		{
			name: "Name wrong",
			f:    func(change *OpenShiftCluster) { change.Name = "wrong" },
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeMismatchingResourceName,
					Message: "The provided resource name 'wrong' did not match the name in the Url 'test-cluster'.",
					Target:  "name",
				},
			},
		},
		{
			name: "ID wrong",
			f:    func(change *OpenShiftCluster) { change.ID = "missmatch" },
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeMismatchingResourceID,
					Message: fmt.Sprintf("The provided resource ID 'missmatch' did not match the name in the Url '/subscriptions/%s/resourcegroups/test-cluster/providers/Microsoft.RedHatOpenShift/openShiftClusters/test-cluster'.", subID),
					Target:  "id",
				},
			},
		},
		{
			name: "Type wrong",
			f:    func(change *OpenShiftCluster) { change.Type = "wrong" },
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeMismatchingResourceType,
					Message: "The provided resource type 'wrong' did not match the name in the Url 'Microsoft.RedHatOpenShift/openShiftClusters'.",
					Target:  "type",
				},
			},
		},
	}
	v := &validator{
		location:   testLocation,
		resourceID: resID,
		r: azure.Resource{
			SubscriptionID: subID,
			ResourceGroup:  clusterName,
			Provider:       "Microsoft.RedHatOpenShift",
			ResourceType:   "openShiftClusters",
			ResourceName:   clusterName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oc := goodOC.DeepCopy()
			tt.f(oc)
			err := v.validateOpenShiftCluster(oc)
			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("OpenShiftCluster.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPropertiesValidate(t *testing.T) {
	tests := []validateTest{
		{
			name: "pass",
			f:    func(change *OpenShiftCluster) {},
		},
		{
			name: "apiServerURL invalid",
			f:    func(change *OpenShiftCluster) { change.Properties.APIServerURL = `f://\\` },
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidParameter,
					Message: `The provided API server URL 'f://\\' is invalid.`,
					Target:  ".test.apiserverURL",
				},
			},
		},
		{
			name: "ConsoleURL invalid",
			f:    func(change *OpenShiftCluster) { change.Properties.ConsoleURL = `f://\\` },
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidParameter,
					Message: `The provided console URL 'f://\\' is invalid.`,
					Target:  ".test.consoleURL",
				},
			},
		},
		{
			name: "wrong provisioning state",
			f:    func(change *OpenShiftCluster) { change.Properties.ProvisioningState = "waat" },
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidParameter,
					Message: "The provided provisioning state 'waat' is invalid.",
					Target:  ".test.provisioningState",
				},
			},
		},
	}
	v := &validator{
		location:   testLocation,
		resourceID: resID,
		r: azure.Resource{
			SubscriptionID: subID,
			ResourceGroup:  clusterName,
			Provider:       "Microsoft.RedHatOpenShift",
			ResourceType:   "openShiftClusters",
			ResourceName:   clusterName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oc := goodOC.DeepCopy()
			tt.f(oc)
			err := v.validateProperties(".test", &oc.Properties)
			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("Properties.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServicePrincipalProfileValidate(t *testing.T) {
	tests := []validateTest{
		{
			name: "good",
			f:    func(change *OpenShiftCluster) {},
		},
		{
			name: "no secret",
			f: func(change *OpenShiftCluster) {
				change.Properties.ServicePrincipalProfile.ClientSecret = ""
			},
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidParameter,
					Message: "The provided client secret is invalid.",
					Target:  ".test.clientSecret",
				},
			},
		},
		{
			name: "invalid clientID",
			f: func(change *OpenShiftCluster) {
				change.Properties.ServicePrincipalProfile.ClientID = "not a uuid"
			},
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidParameter,
					Message: "The provided client ID 'not a uuid' is invalid.",
					Target:  ".test.clientId",
				},
			},
		},
	}
	v := &validator{
		location:   testLocation,
		resourceID: resID,
		r: azure.Resource{
			SubscriptionID: subID,
			ResourceGroup:  clusterName,
			Provider:       "Microsoft.RedHatOpenShift",
			ResourceType:   "openShiftClusters",
			ResourceName:   clusterName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oc := goodOC.DeepCopy()
			tt.f(oc)
			err := v.validateServicePrincipalProfile(".test", &oc.Properties.ServicePrincipalProfile)
			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("ServicePrincipalProfile.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNetworkProfileValidate(t *testing.T) {
	tests := []validateTest{
		{
			name: "pass",
			f:    func(change *OpenShiftCluster) {},
		},
		{
			name: "podCIDR invalid",
			f:    func(change *OpenShiftCluster) { change.Properties.NetworkProfile.PodCIDR = "not a CIDR" },
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidParameter,
					Message: "The provided pod CIDR 'not a CIDR' is invalid: 'invalid CIDR address: not a CIDR'.",
					Target:  ".test.podCidr",
				},
			},
		},
		{
			name: "serviceCIDR invalid",
			f:    func(change *OpenShiftCluster) { change.Properties.NetworkProfile.ServiceCIDR = "not a CIDR" },
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidParameter,
					Message: "The provided service CIDR 'not a CIDR' is invalid: 'invalid CIDR address: not a CIDR'.",
					Target:  ".test.serviceCidr",
				},
			},
		},
		{
			name: "podCIDR too small",
			f:    func(change *OpenShiftCluster) { change.Properties.NetworkProfile.PodCIDR = "10.0.0.0/24" },
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidParameter,
					Message: "The provided vnet CIDR '10.0.0.0/24' is invalid: must be /18 or larger.",
					Target:  ".test.podCidr",
				},
			},
		},
		{
			name: "serviceCIDR too small",
			f:    func(change *OpenShiftCluster) { change.Properties.NetworkProfile.ServiceCIDR = "10.0.0.0/24" },
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidParameter,
					Message: "The provided vnet CIDR '10.0.0.0/24' is invalid: must be /22 or larger.",
					Target:  ".test.serviceCidr",
				},
			},
		},
	}
	v := &validator{
		location:   testLocation,
		resourceID: resID,
		r: azure.Resource{
			SubscriptionID: subID,
			ResourceGroup:  clusterName,
			Provider:       "Microsoft.RedHatOpenShift",
			ResourceType:   "openShiftClusters",
			ResourceName:   clusterName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oc := goodOC.DeepCopy()
			tt.f(oc)
			err := v.validateNetworkProfile(".test", &oc.Properties.NetworkProfile)
			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("NetworkProfile.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMasterProfileValidate(t *testing.T) {
	sub2ID := uuid.NewV4().String()
	resID := fmt.Sprintf("/subscriptions/%s/resourcegroups/test-vnet/providers/Microsoft.Network/virtualNetworks/test-vnet", subID)
	subnet2ID := fmt.Sprintf("/subscriptions/%s/resourcegroups/test-vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", sub2ID)
	tests := []validateTest{
		{
			name: "pass",
			f:    func(change *OpenShiftCluster) {},
		},
		{
			name: "invalid vmsize",
			f:    func(change *OpenShiftCluster) { change.Properties.MasterProfile.VMSize = VMSizeStandardD2sV3 },
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidParameter,
					Message: "The provided master VM size 'Standard_D2s_v3' is invalid.",
					Target:  ".test.vmSize",
				},
			},
		},
		{
			name: "invalid subnetid",
			f:    func(change *OpenShiftCluster) { change.Properties.MasterProfile.SubnetID = "not right" },
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidParameter,
					Message: "The provided master VM subnet 'not right' is invalid.",
					Target:  ".test.subnetId",
				},
			},
		},
		{
			name: "subs not matching",
			f:    func(change *OpenShiftCluster) { change.Properties.MasterProfile.SubnetID = subnet2ID },
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidParameter,
					Message: fmt.Sprintf("The provided master VM subnet '/subscriptions/%s/resourcegroups/test-vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master' is invalid: must be in same subscription as cluster.", sub2ID),
					Target:  ".test.subnetId",
				},
			},
		},
	}

	v := &validator{
		location:   testLocation,
		resourceID: resID,
		r: azure.Resource{
			SubscriptionID: subID,
			ResourceGroup:  clusterName,
			Provider:       "Microsoft.RedHatOpenShift",
			ResourceType:   "openShiftClusters",
			ResourceName:   clusterName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testOC := goodOC.DeepCopy()
			tt.f(testOC)
			err := v.validateMasterProfile(".test", &testOC.Properties.MasterProfile)
			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("MasterProfile.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWorkerProfileValidate(t *testing.T) {
	subnet2ID := fmt.Sprintf("/subscriptions/%s/resourcegroups/test-vnet/providers/Microsoft.Network/virtualNetworks/other-vnet/subnets/other", subID)
	tests := []validateTest{
		{
			name: "pass",
			f:    func(oc *OpenShiftCluster) {},
		},
		{
			name: "disk too small",
			f:    func(oc *OpenShiftCluster) { oc.Properties.WorkerProfiles[0].DiskSizeGB = 100 },
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidParameter,
					Message: "The provided worker disk size '100' is invalid.",
					Target:  ".test.diskSizeGB",
				},
			},
		},
		{
			name: "count too small",
			f:    func(oc *OpenShiftCluster) { oc.Properties.WorkerProfiles[0].Count = 2 },
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidParameter,
					Message: "The provided worker count '2' is invalid.",
					Target:  ".test.count",
				},
			},
		},
		{
			name: "count too big",
			f:    func(oc *OpenShiftCluster) { oc.Properties.WorkerProfiles[0].Count = 21 },
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidParameter,
					Message: "The provided worker count '21' is invalid.",
					Target:  ".test.count",
				},
			},
		},
		{
			name: "subnet different to master",
			f:    func(oc *OpenShiftCluster) { oc.Properties.WorkerProfiles[0].SubnetID = subnet2ID },
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidParameter,
					Message: fmt.Sprintf("The provided worker VM subnet '/subscriptions/%s/resourcegroups/test-vnet/providers/Microsoft.Network/virtualNetworks/other-vnet/subnets/other' is invalid: must be in the same vnet as master VM subnet '/subscriptions/%s/resourcegroups/test-vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master'", subID, subID),
					Target:  ".test.subnetId",
				},
			},
		},
		{
			name: "must be worker",
			f:    func(oc *OpenShiftCluster) { oc.Properties.WorkerProfiles[0].Name = "buzzy-bee" },
			wantErr: &api.CloudError{
				StatusCode: 400,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidParameter,
					Message: "The provided worker name 'buzzy-bee' is invalid.",
					Target:  ".test.name",
				},
			},
		},
	}
	v := &validator{
		location:   testLocation,
		resourceID: resID,
		r: azure.Resource{
			SubscriptionID: subID,
			ResourceGroup:  clusterName,
			Provider:       "Microsoft.RedHatOpenShift",
			ResourceType:   "openShiftClusters",
			ResourceName:   clusterName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testOC := goodOC.DeepCopy()
			tt.f(testOC)
			err := v.validateWorkerProfile(".test", &testOC.Properties.WorkerProfiles[0], &goodOC.Properties.MasterProfile)
			if !reflect.DeepEqual(err, tt.wantErr) {
				t.Errorf("WorkerProfile.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
