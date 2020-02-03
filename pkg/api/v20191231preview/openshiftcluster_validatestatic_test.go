package v20191231preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/go-autorest/autorest/azure"
	uuid "github.com/satori/go.uuid"

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

	v = &openShiftClusterValidator{
		sv: openShiftClusterStaticValidator{
			location:   "location",
			resourceID: id,
			r: azure.Resource{
				SubscriptionID: subscriptionID,
				ResourceGroup:  "resourceGroup",
				Provider:       "Microsoft.RedHatOpenShift",
				ResourceType:   "openshiftClusters",
				ResourceName:   "resourceName",
			},
		},
	}
)

func validOpenShiftCluster() *OpenShiftCluster {
	oc := exampleOpenShiftCluster()
	oc.ID = id
	oc.Properties.ClusterProfile.ResourceGroupID = fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", subscriptionID)
	oc.Properties.ServicePrincipalProfile.ClientID = "2b5ba2c6-6205-4fc4-8b5d-9fea369ae1a2"
	oc.Properties.MasterProfile.SubnetID = fmt.Sprintf("/subscriptions/%s/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", subscriptionID)
	oc.Properties.WorkerProfiles[0].SubnetID = fmt.Sprintf("/subscriptions/%s/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/worker", subscriptionID)

	return oc
}

func runTests(t *testing.T, tests []*validateTest, delta bool) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oc := validOpenShiftCluster()
			if tt.modify != nil {
				tt.modify(oc)
			}

			current := &api.OpenShiftCluster{}
			if delta {
				(&openShiftClusterConverter{}).ToInternal(validOpenShiftCluster(), current)
			} else {
				(&openShiftClusterConverter{}).ToInternal(oc, current)
			}
			err := v.Static(oc, current)
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

func TestOpenShiftClusterStaticValidate(t *testing.T) {
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

	runTests(t, tests, false)
}

func TestOpenShiftClusterStaticValidateProperties(t *testing.T) {
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
	}

	runTests(t, tests, false)
}

func TestOpenShiftClusterStaticValidateClusterProfile(t *testing.T) {
	tests := []*validateTest{
		{
			name: "valid",
		},
		{
			name: "empty domain invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = ""
			},
			wantErr: "400: InvalidParameter: properties.clusterProfile.domain: The provided domain '' is invalid.",
		},
		{
			name: "upper case domain invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "BAD"
			},
			wantErr: "400: InvalidParameter: properties.clusterProfile.domain: The provided domain 'BAD' is invalid.",
		},
		{
			name: "domain invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "!"
			},
			wantErr: "400: InvalidParameter: properties.clusterProfile.domain: The provided domain '!' is invalid.",
		},
		{
			name: "version invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterProfile.Version = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.clusterProfile.version: The provided version 'invalid' is invalid.",
		},
		{
			name: "resourceGroupId invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterProfile.ResourceGroupID = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.clusterProfile.resourceGroupId: The provided resource group 'invalid' is invalid.",
		},
		{
			name: "cluster resource group subscriptionId not matching cluster subscriptionId",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterProfile.ResourceGroupID = "/subscriptions/7a3036d1-60a1-4605-8a41-44955e050804/resourcegroups/test-cluster"
			},
			wantErr: "400: InvalidParameter: properties.clusterProfile.resourceGroupId: The provided resource group '/subscriptions/7a3036d1-60a1-4605-8a41-44955e050804/resourcegroups/test-cluster' is invalid: must be in same subscription as cluster.",
		},
	}

	runTests(t, tests, false)
}

func TestOpenShiftClusterStaticValidateConsoleProfile(t *testing.T) {
	tests := []*validateTest{
		{
			name: "valid",
		},
		{
			name: "empty console url valid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ConsoleProfile.URL = ""
			},
		},
		{
			name: "console url invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ConsoleProfile.URL = "\x00"
			},
			wantErr: "400: InvalidParameter: properties.consoleProfile.url: The provided console URL '\x00' is invalid.",
		},
	}

	runTests(t, tests, false)

}

func TestOpenShiftClusterStaticValidateServicePrincipalProfile(t *testing.T) {
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

	runTests(t, tests, false)
}

func TestOpenShiftClusterStaticValidateNetworkProfile(t *testing.T) {
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

	runTests(t, tests, false)
}

func TestOpenShiftClusterStaticValidateMasterProfile(t *testing.T) {
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

	runTests(t, tests, false)
}

func TestOpenShiftClusterStaticValidateWorkerProfile(t *testing.T) {
	tests := []*validateTest{
		{
			name: "valid",
		},
		{
			name: "name invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].Name = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles['invalid'].name: The provided worker name 'invalid' is invalid.",
		},
		{
			name: "vmSize invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].VMSize = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles['worker'].vmSize: The provided worker VM size 'invalid' is invalid.",
		},
		{
			name: "disk too small",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].DiskSizeGB = 127
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles['worker'].diskSizeGB: The provided worker disk size '127' is invalid.",
		},
		{
			name: "subnetId invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].SubnetID = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles['worker'].subnetId: The provided worker VM subnet 'invalid' is invalid.",
		},
		{
			name: "master and worker subnets not in same vnet",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].SubnetID = fmt.Sprintf("/subscriptions/%s/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/different-vnet/subnets/worker", subscriptionID)
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles['worker'].subnetId: The provided worker VM subnet '/subscriptions/af848f0a-dbe3-449f-9ccd-6f23ac6ef9f1/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/different-vnet/subnets/worker' is invalid: must be in the same vnet as master VM subnet '/subscriptions/af848f0a-dbe3-449f-9ccd-6f23ac6ef9f1/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master'.",
		},
		{
			name: "master and worker subnets not different",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].SubnetID = oc.Properties.MasterProfile.SubnetID
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles['worker'].subnetId: The provided worker VM subnet '/subscriptions/af848f0a-dbe3-449f-9ccd-6f23ac6ef9f1/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master' is invalid: must be different to master VM subnet '/subscriptions/af848f0a-dbe3-449f-9ccd-6f23ac6ef9f1/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master'.",
		},
		{
			name: "count too small",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].Count = 2
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles['worker'].count: The provided worker count '2' is invalid.",
		},
		{
			name: "count too big",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].Count = 21
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles['worker'].count: The provided worker count '21' is invalid.",
		},
	}

	runTests(t, tests, false)
}

func TestOpenShiftClusterStaticValidateAPIServerProfile(t *testing.T) {
	tests := []*validateTest{
		{
			name: "valid",
		},
		{
			name: "visibility invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.APIServerProfile.Visibility = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.apiserverProfile.visibility: The provided visibility 'invalid' is invalid.",
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

	runTests(t, tests, false)
}

func TestOpenShiftClusterStaticValidateIngressProfile(t *testing.T) {
	tests := []*validateTest{
		{
			name: "valid",
		},
		{
			name: "name invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.IngressProfiles[0].Name = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.ingressProfiles['invalid'].name: The provided ingress name 'invalid' is invalid.",
		},
		{
			name: "visibility invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.IngressProfiles[0].Visibility = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.ingressProfiles['default'].visibility: The provided visibility 'invalid' is invalid.",
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
			wantErr: "400: InvalidParameter: properties.ingressProfiles['default'].ip: The provided IP 'invalid' is invalid.",
		},
		{
			name: "ipv6 ip invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.IngressProfiles[0].IP = "::"
			},
			wantErr: "400: InvalidParameter: properties.ingressProfiles['default'].ip: The provided IP '::' is invalid: must be IPv4.",
		},
	}

	runTests(t, tests, false)
}

func TestOpenShiftClusterStaticValidateDelta(t *testing.T) {
	tests := []*validateTest{
		{
			name: "valid",
		},
		{
			name:   "valid id case change",
			modify: func(oc *OpenShiftCluster) { oc.ID = strings.ToUpper(oc.ID) },
		},
		{
			name:   "valid name case change",
			modify: func(oc *OpenShiftCluster) { oc.Name = strings.ToUpper(oc.Name) },
		},
		{
			name:   "valid type case change",
			modify: func(oc *OpenShiftCluster) { oc.Type = strings.ToUpper(oc.Type) },
		},
		{
			name:    "location change",
			modify:  func(oc *OpenShiftCluster) { oc.Location = strings.ToUpper(oc.Location) },
			wantErr: "400: PropertyChangeNotAllowed: location: Changing property 'location' is not allowed.",
		},
		{
			name:   "valid tags change",
			modify: func(oc *OpenShiftCluster) { oc.Tags = Tags{"new": "value"} },
		},
		{
			name:    "provisioningState change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ProvisioningState = ProvisioningStateFailed },
			wantErr: "400: PropertyChangeNotAllowed: properties.provisioningState: Changing property 'properties.provisioningState' is not allowed.",
		},
		{
			name:    "console url change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ConsoleProfile.URL = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.consoleProfile.url: Changing property 'properties.consoleProfile.url' is not allowed.",
		},
		{
			name:    "domain change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ClusterProfile.Domain = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.clusterProfile.domain: Changing property 'properties.clusterProfile.domain' is not allowed.",
		},
		{
			name:    "version change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ClusterProfile.Version = "" },
			wantErr: "400: PropertyChangeNotAllowed: properties.clusterProfile.version: Changing property 'properties.clusterProfile.version' is not allowed.",
		},
		{
			name: "resource group change",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterProfile.ResourceGroupID = oc.Properties.ClusterProfile.ResourceGroupID[:strings.LastIndexByte(oc.Properties.ClusterProfile.ResourceGroupID, '/')] + "/changed"
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.clusterProfile.resourceGroupId: Changing property 'properties.clusterProfile.resourceGroupId' is not allowed.",
		},
		{
			name: "apiServer private change",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.APIServerProfile.Visibility = VisibilityPrivate
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.apiserverProfile.visibility: Changing property 'properties.apiserverProfile.visibility' is not allowed.",
		},
		{
			name:    "apiServer url change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.APIServerProfile.URL = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.apiserverProfile.url: Changing property 'properties.apiserverProfile.url' is not allowed.",
		},
		{
			name:    "apiServer ip change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.APIServerProfile.IP = "2.3.4.5" },
			wantErr: "400: PropertyChangeNotAllowed: properties.apiserverProfile.ip: Changing property 'properties.apiserverProfile.ip' is not allowed.",
		},
		{
			name: "ingress private change",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.IngressProfiles[0].Visibility = VisibilityPrivate
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.ingressProfiles['default'].visibility: Changing property 'properties.ingressProfiles['default'].visibility' is not allowed.",
		},
		{
			name:    "ingress ip change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.IngressProfiles[0].IP = "2.3.4.5" },
			wantErr: "400: PropertyChangeNotAllowed: properties.ingressProfiles['default'].ip: Changing property 'properties.ingressProfiles['default'].ip' is not allowed.",
		},
		{
			name:    "clientId change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ServicePrincipalProfile.ClientID = uuid.NewV4().String() },
			wantErr: "400: PropertyChangeNotAllowed: properties.servicePrincipalProfile.clientId: Changing property 'properties.servicePrincipalProfile.clientId' is not allowed.",
		},
		{
			name:    "clientSecret change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ServicePrincipalProfile.ClientSecret = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.servicePrincipalProfile.clientSecret: Changing property 'properties.servicePrincipalProfile.clientSecret' is not allowed.",
		},
		{
			name:    "podCidr change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.NetworkProfile.PodCIDR = "0.0.0.0/0" },
			wantErr: "400: PropertyChangeNotAllowed: properties.networkProfile.podCidr: Changing property 'properties.networkProfile.podCidr' is not allowed.",
		},
		{
			name:    "serviceCidr change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.NetworkProfile.ServiceCIDR = "0.0.0.0/0" },
			wantErr: "400: PropertyChangeNotAllowed: properties.networkProfile.serviceCidr: Changing property 'properties.networkProfile.serviceCidr' is not allowed.",
		},
		{
			name: "master subnetId change",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.MasterProfile.SubnetID = oc.Properties.MasterProfile.SubnetID[:strings.LastIndexByte(oc.Properties.MasterProfile.SubnetID, '/')] + "/changed"
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.masterProfile.subnetId: Changing property 'properties.masterProfile.subnetId' is not allowed.",
		},
		{
			name:    "worker vmSize change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.WorkerProfiles[0].VMSize = VMSizeStandardD4sV3 },
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles['worker'].vmSize: Changing property 'properties.workerProfiles['worker'].vmSize' is not allowed.",
		},
		{
			name:    "worker diskSizeGB change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.WorkerProfiles[0].DiskSizeGB++ },
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles['worker'].diskSizeGB: Changing property 'properties.workerProfiles['worker'].diskSizeGB' is not allowed.",
		},
		{
			name: "worker subnetId change",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].SubnetID = oc.Properties.WorkerProfiles[0].SubnetID[:strings.LastIndexByte(oc.Properties.WorkerProfiles[0].SubnetID, '/')] + "/changed"
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles['worker'].subnetId: Changing property 'properties.workerProfiles['worker'].subnetId' is not allowed.",
		},
		{
			name:    "workerProfiles count change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.WorkerProfiles[0].Count++ },
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles['worker'].count: Changing property 'properties.workerProfiles['worker'].count' is not allowed.",
		},
	}

	runTests(t, tests, true)
}
