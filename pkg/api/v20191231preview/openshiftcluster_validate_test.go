package v20191231preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/test/validate"
)

type validateTest struct {
	name    string
	modify  func(oc *OpenShiftCluster)
	wantErr string
}

var (
	subscriptionID = "af848f0a-dbe3-449f-9ccd-6f23ac6ef9f1"
	id             = fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/microsoft.redhatopenshift/openshiftclusters/resourceName", subscriptionID)

	v = &validator{
		location:   "location",
		resourceID: id,
		r: azure.Resource{
			SubscriptionID: subscriptionID,
			ResourceGroup:  "resourceGroup",
			Provider:       "Microsoft.RedHatOpenShift",
			ResourceType:   "openshiftClusters",
			ResourceName:   "resourceName",
		},
	}
)

func validOpenShiftCluster() *OpenShiftCluster {
	oc := exampleOpenShiftCluster()
	oc.ID = id
	oc.Properties.ServicePrincipalProfile.ClientID = "2b5ba2c6-6205-4fc4-8b5d-9fea369ae1a2"
	oc.Properties.MasterProfile.SubnetID = fmt.Sprintf("/subscriptions/%s/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", subscriptionID)
	oc.Properties.WorkerProfiles[0].SubnetID = fmt.Sprintf("/subscriptions/%s/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/worker", subscriptionID)

	return oc
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

				cloudErr := err.(*api.CloudError)

				if cloudErr.StatusCode != http.StatusBadRequest {
					t.Error(cloudErr.StatusCode)
				}
				if cloudErr.Target == "" {
					t.Error("target is required")
				}

				validate.CloudError(t, err)
			}
		})
	}
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
			wantErr: "400: MismatchingResourceID: id: The provided resource ID 'wrong' did not match the name in the Url '/subscriptions/af848f0a-dbe3-449f-9ccd-6f23ac6ef9f1/resourcegroups/resourceGroup/providers/microsoft.redhatopenshift/openshiftclusters/resourceName'.",
		},
		{
			name: "name wrong",
			modify: func(oc *OpenShiftCluster) {
				oc.Name = "wrong"
			},
			wantErr: "400: MismatchingResourceName: name: The provided resource name 'wrong' did not match the name in the Url 'resourceName'.",
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
			name: "empty clusterDomain invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterDomain = ""
			},
			wantErr: "400: InvalidParameter: properties.clusterDomain: The provided cluster domain '' is invalid.",
		},
		{
			name: "upper case clusterDomain invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterDomain = "BAD"
			},
			wantErr: "400: InvalidParameter: properties.clusterDomain: The provided cluster domain 'BAD' is invalid.",
		},
		{
			name: "clusterDomain invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterDomain = "!"
			},
			wantErr: "400: InvalidParameter: properties.clusterDomain: The provided cluster domain '!' is invalid.",
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

func TestValidateAPIServerProfile(t *testing.T) {
	tests := []*validateTest{
		{
			name: "valid",
		},
		{
			name: "empty url valid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.APIServerProfile.URL = ""
			},
		},
		{
			name: "url invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.APIServerProfile.URL = "\x00"
			},
			wantErr: "400: InvalidParameter: properties.apiserverProfile.url: The provided URL '\x00' is invalid.",
		},
		{
			name: "empty ip valid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.APIServerProfile.IP = ""
			},
		},
		{
			name: "ip invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.APIServerProfile.IP = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.apiserverProfile.ip: The provided IP 'invalid' is invalid.",
		},
		{
			name: "ipv6 ip invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.APIServerProfile.IP = "::"
			},
			wantErr: "400: InvalidParameter: properties.apiserverProfile.ip: The provided IP '::' is invalid: must be IPv4.",
		},
	}

	runTests(t, tests, func(oc *OpenShiftCluster) error {
		return v.validateAPIServerProfile("properties.apiserverProfile", &oc.Properties.APIServerProfile)
	})
}

func TestValidateIngressProfile(t *testing.T) {
	tests := []*validateTest{
		{
			name: "valid",
		},
		{
			name: "name invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.IngressProfiles[0].Name = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.ingressProfiles[0].name: The provided ingress name 'invalid' is invalid.",
		},
		{
			name: "empty ip valid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.IngressProfiles[0].IP = ""
			},
		},
		{
			name: "ip invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.IngressProfiles[0].IP = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.ingressProfiles[0].ip: The provided IP 'invalid' is invalid.",
		},
		{
			name: "ipv6 ip invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.IngressProfiles[0].IP = "::"
			},
			wantErr: "400: InvalidParameter: properties.ingressProfiles[0].ip: The provided IP '::' is invalid: must be IPv4.",
		},
	}

	runTests(t, tests, func(oc *OpenShiftCluster) error {
		return v.validateIngressProfile("properties.ingressProfiles[0]", &oc.Properties.IngressProfiles[0])
	})
}

func TestOpenShiftClusterValidateDelta(t *testing.T) {
	tests := []*validateTest{
		{
			name: "valid",
		},
		{
			name:   "valid id case change",
			modify: func(oc *OpenShiftCluster) { oc.ID = strings.ToUpper(oc.ID) },
		},
		{
			name:    "id change",
			modify:  func(oc *OpenShiftCluster) { oc.ID = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: id: Changing property 'id' is not allowed.",
		},
		{
			name:   "valid name case change",
			modify: func(oc *OpenShiftCluster) { oc.Name = strings.ToUpper(oc.Name) },
		},
		{
			name:    "name change",
			modify:  func(oc *OpenShiftCluster) { oc.Name = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: name: Changing property 'name' is not allowed.",
		},
		{
			name:   "valid type case change",
			modify: func(oc *OpenShiftCluster) { oc.Type = strings.ToUpper(oc.Type) },
		},
		{
			name:    "type change",
			modify:  func(oc *OpenShiftCluster) { oc.Type = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: type: Changing property 'type' is not allowed.",
		},
		{
			name:    "location change",
			modify:  func(oc *OpenShiftCluster) { oc.Location = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: location: Changing property 'location' is not allowed.",
		},
		{
			name:   "valid tags change",
			modify: func(oc *OpenShiftCluster) { oc.Tags = Tags{"new": "value"} },
		},
		{
			name:    "provisioningState change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ProvisioningState = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.provisioningState: Changing property 'properties.provisioningState' is not allowed.",
		},
		{
			name:    "clusterDomain change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ClusterDomain = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.clusterDomain: Changing property 'properties.clusterDomain' is not allowed.",
		},
		{
			name: "apiServer private change",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.APIServerProfile.Private = !oc.Properties.APIServerProfile.Private
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.apiserverProfile.private: Changing property 'properties.apiserverProfile.private' is not allowed.",
		},
		{
			name:    "apiServer url change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.APIServerProfile.URL = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.apiserverProfile.url: Changing property 'properties.apiserverProfile.url' is not allowed.",
		},
		{
			name:    "apiServer ip change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.APIServerProfile.IP = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.apiserverProfile.ip: Changing property 'properties.apiserverProfile.ip' is not allowed.",
		},
		{
			name:    "ingress name change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.IngressProfiles[0].Name = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.ingressProfiles[0].name: Changing property 'properties.ingressProfiles[0].name' is not allowed.",
		},
		{
			name: "ingress private change",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.IngressProfiles[0].Private = !oc.Properties.IngressProfiles[0].Private
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.ingressProfiles[0].private: Changing property 'properties.ingressProfiles[0].private' is not allowed.",
		},
		{
			name:    "ingress ip change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.IngressProfiles[0].IP = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.ingressProfiles[0].ip: Changing property 'properties.ingressProfiles[0].ip' is not allowed.",
		},
		{
			name:    "consoleUrl change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ConsoleURL = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.consoleUrl: Changing property 'properties.consoleUrl' is not allowed.",
		},
		{
			name:    "clientId change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ServicePrincipalProfile.ClientID = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.servicePrincipalProfile.clientId: Changing property 'properties.servicePrincipalProfile.clientId' is not allowed.",
		},
		{
			name:    "clientSecret change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ServicePrincipalProfile.ClientSecret = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.servicePrincipalProfile.clientSecret: Changing property 'properties.servicePrincipalProfile.clientSecret' is not allowed.",
		},
		{
			name:    "podCidr change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.NetworkProfile.PodCIDR = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.networkProfile.podCidr: Changing property 'properties.networkProfile.podCidr' is not allowed.",
		},
		{
			name:    "serviceCidr change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.NetworkProfile.ServiceCIDR = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.networkProfile.serviceCidr: Changing property 'properties.networkProfile.serviceCidr' is not allowed.",
		},
		{
			name:    "master vmSize change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.MasterProfile.VMSize = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.masterProfile.vmSize: Changing property 'properties.masterProfile.vmSize' is not allowed.",
		},
		{
			name:    "master subnetId change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.MasterProfile.SubnetID = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.masterProfile.subnetId: Changing property 'properties.masterProfile.subnetId' is not allowed.",
		},
		{
			name:    "worker name change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.WorkerProfiles[0].Name = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles[0].name: Changing property 'properties.workerProfiles[0].name' is not allowed.",
		},
		{
			name:    "worker vmSize change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.WorkerProfiles[0].VMSize = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles[0].vmSize: Changing property 'properties.workerProfiles[0].vmSize' is not allowed.",
		},
		{
			name:    "worker diskSizeGB change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.WorkerProfiles[0].DiskSizeGB++ },
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles[0].diskSizeGB: Changing property 'properties.workerProfiles[0].diskSizeGB' is not allowed.",
		},
		{
			name:    "worker subnetId change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.WorkerProfiles[0].SubnetID = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles[0].subnetId: Changing property 'properties.workerProfiles[0].subnetId' is not allowed.",
		},
		{
			name: "additional workerProfile",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles = append(oc.Properties.WorkerProfiles, WorkerProfile{})
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles: Changing property 'properties.workerProfiles' is not allowed.",
		},
		{
			name:   "valid count change",
			modify: func(oc *OpenShiftCluster) { oc.Properties.WorkerProfiles[0].Count++ },
		},
	}

	current := validOpenShiftCluster()
	runTests(t, tests, func(oc *OpenShiftCluster) error {
		return v.validateOpenShiftClusterDelta(oc, current)
	})
}
