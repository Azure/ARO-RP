package v20210901preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
	"github.com/Azure/ARO-RP/pkg/util/version"
	"github.com/Azure/ARO-RP/test/validate"
)

type validateTest struct {
	name                string
	current             func(oc *OpenShiftCluster)
	modify              func(oc *OpenShiftCluster)
	requireD2sV3Workers bool
	wantErr             string
}

type testMode string

const (
	testModeCreate testMode = "Create"
	testModeUpdate testMode = "Update"
)

var (
	subscriptionID = "00000000-0000-0000-0000-000000000000"
	id             = fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/microsoft.redhatopenshift/openshiftclusters/resourceName", subscriptionID)
)

func validOpenShiftCluster() *OpenShiftCluster {
	timestamp, err := time.Parse(time.RFC3339, "2021-01-23T12:34:54.0000000Z")
	if err != nil {
		panic(err)
	}

	oc := &OpenShiftCluster{
		ID:       id,
		Name:     "resourceName",
		Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
		Location: "location",
		Tags: Tags{
			"key": "value",
		},
		SystemData: &SystemData{
			CreatedBy:          "00000000-0000-0000-0000-000000000000",
			CreatedByType:      CreatedByTypeApplication,
			CreatedAt:          &timestamp,
			LastModifiedBy:     "00000000-0000-0000-0000-000000000000",
			LastModifiedByType: CreatedByTypeApplication,
			LastModifiedAt:     &timestamp,
		},
		Properties: OpenShiftClusterProperties{
			ProvisioningState: ProvisioningStateSucceeded,
			ClusterProfile: ClusterProfile{
				PullSecret:      `{"auths":{"registry.connect.redhat.com":{"auth":""},"registry.redhat.io":{"auth":""}}}`,
				Domain:          "cluster.location.aroapp.io",
				Version:         version.DefaultInstallStream.Version.String(),
				ResourceGroupID: fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", subscriptionID),
			},
			ConsoleProfile: ConsoleProfile{
				URL: "https://console-openshift-console.apps.cluster.location.aroapp.io/",
			},
			ServicePrincipalProfile: ServicePrincipalProfile{
				ClientSecret: "clientSecret",
				ClientID:     "11111111-1111-1111-1111-111111111111",
			},
			NetworkProfile: NetworkProfile{
				PodCIDR:                "10.128.0.0/14",
				ServiceCIDR:            "172.30.0.0/16",
				SoftwareDefinedNetwork: SoftwareDefinedNetworkOVNKubernetes,
			},
			MasterProfile: MasterProfile{
				VMSize:           VMSizeStandardD8sV3,
				EncryptionAtHost: EncryptionAtHostDisabled,
				SubnetID:         fmt.Sprintf("/subscriptions/%s/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", subscriptionID),
			},
			WorkerProfiles: []WorkerProfile{
				{
					Name:             "worker",
					VMSize:           VMSizeStandardD4sV3,
					EncryptionAtHost: EncryptionAtHostDisabled,
					DiskSizeGB:       128,
					SubnetID:         fmt.Sprintf("/subscriptions/%s/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/worker", subscriptionID),
					Count:            3,
				},
			},
			APIServerProfile: APIServerProfile{
				Visibility: VisibilityPublic,
				URL:        "https://api.cluster.location.aroapp.io:6443/",
				IP:         "1.2.3.4",
			},
			IngressProfiles: []IngressProfile{
				{
					Name:       "default",
					Visibility: VisibilityPublic,
					IP:         "1.2.3.4",
				},
			},
		},
	}

	return oc
}

func runTests(t *testing.T, mode testMode, tests []*validateTest) {
	t.Run(string(mode), func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				v := &openShiftClusterStaticValidator{
					location:            "location",
					domain:              "location.aroapp.io",
					requireD2sV3Workers: tt.requireD2sV3Workers,
					resourceID:          id,
					r: azure.Resource{
						SubscriptionID: subscriptionID,
						ResourceGroup:  "resourceGroup",
						Provider:       "Microsoft.RedHatOpenShift",
						ResourceType:   "openshiftClusters",
						ResourceName:   "resourceName",
					},
				}

				validOCForTest := func() *OpenShiftCluster {
					oc := validOpenShiftCluster()
					if tt.current != nil {
						tt.current(oc)
					}
					return oc
				}

				oc := validOCForTest()
				if tt.modify != nil {
					tt.modify(oc)
				}

				var current *api.OpenShiftCluster
				if mode == testModeUpdate {
					current = &api.OpenShiftCluster{}
					(&openShiftClusterConverter{}).ToInternal(validOCForTest(), current)
				}

				err := v.Static(oc, current, v.location, v.domain, tt.requireD2sV3Workers, v.resourceID)
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
	})
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
			wantErr: "400: MismatchingResourceID: id: The provided resource ID 'wrong' did not match the name in the Url '/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup/providers/microsoft.redhatopenshift/openshiftclusters/resourceName'.",
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

	runTests(t, testModeCreate, tests)
	runTests(t, testModeUpdate, tests)
}

func TestOpenShiftClusterStaticValidateProperties(t *testing.T) {
	commonTests := []*validateTest{
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
	}
	createTests := []*validateTest{
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

	runTests(t, testModeCreate, createTests)
	runTests(t, testModeCreate, commonTests)
	runTests(t, testModeUpdate, commonTests)
}

func TestOpenShiftClusterStaticValidateClusterProfile(t *testing.T) {
	commonTests := []*validateTest{
		{
			name: "valid",
		},
		{
			name: "pull secret not a map",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterProfile.PullSecret = "1"
			},
			wantErr: "400: InvalidParameter: properties.clusterProfile.pullSecret: The provided pull secret is invalid.",
		},
		{
			name: "pull secret invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterProfile.PullSecret = "{"
			},
			wantErr: "400: InvalidParameter: properties.clusterProfile.pullSecret: The provided pull secret is invalid.",
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
			name: "wrong location managed domain invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "cluster.wronglocation.aroapp.io"
			},
			wantErr: "400: InvalidParameter: properties.clusterProfile.domain: The provided domain 'cluster.wronglocation.aroapp.io' is invalid.",
		},
		{
			name: "double part managed domain invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "foo.bar.location.aroapp.io"
			},
			wantErr: "400: InvalidParameter: properties.clusterProfile.domain: The provided domain 'foo.bar.location.aroapp.io' is invalid.",
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
		{
			name: "cluster resourceGroup and external resourceGroup equal",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterProfile.ResourceGroupID = "/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup"
			},
			wantErr: "400: InvalidParameter: properties.clusterProfile.resourceGroupId: The provided resource group '/subscriptions/00000000-0000-0000-0000-000000000000/resourcegroups/resourceGroup' is invalid: must be different from resourceGroup of the OpenShift cluster object.",
		},
	}

	createTests := []*validateTest{
		{
			name: "empty pull secret valid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterProfile.PullSecret = ""
			},
		},
		{
			name: "leading digit domain invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "4k7f9clk"
			},
			wantErr: "400: InvalidParameter: properties.clusterProfile.domain: The provided domain '4k7f9clk' is invalid.",
		},
	}

	updateTests := []*validateTest{
		{
			name: "leading digit domain valid",
			current: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterProfile.Domain = "4k7f9clk"
			},
		},
	}

	runTests(t, testModeCreate, createTests)
	runTests(t, testModeCreate, commonTests)
	runTests(t, testModeUpdate, updateTests)
	runTests(t, testModeUpdate, commonTests)
}

func TestOpenShiftClusterStaticValidateConsoleProfile(t *testing.T) {
	commonTests := []*validateTest{
		{
			name: "valid",
		},
		{
			name: "console url invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ConsoleProfile.URL = "\x00"
			},
			wantErr: "400: InvalidParameter: properties.consoleProfile.url: The provided console URL '\x00' is invalid.",
		},
	}

	createTests := []*validateTest{
		{
			name: "empty console url valid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ConsoleProfile.URL = ""
			},
		},
	}

	runTests(t, testModeCreate, createTests)
	runTests(t, testModeCreate, commonTests)
	runTests(t, testModeUpdate, commonTests)
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

	runTests(t, testModeCreate, tests)
	runTests(t, testModeUpdate, tests)
}

func TestOpenShiftClusterStaticValidateNetworkProfile(t *testing.T) {
	createtests := []*validateTest{
		{
			name: "SoftwareDefinedNetwork create as OpenShiftSDN",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.SoftwareDefinedNetwork = SoftwareDefinedNetworkOpenShiftSDN
			},
		},
	}

	commontests := []*validateTest{
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
		{
			name: "SoftwareDefinedNetwork given as empty",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.SoftwareDefinedNetwork = ""
			},
			wantErr: "400: InvalidParameter: properties.networkProfile.SoftwareDefinedNetwork: The provided SoftwareDefinedNetwork '' is invalid.",
		},
		{
			name: "SoftwareDefinedNetwork given InvalidOption",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.SoftwareDefinedNetwork = "InvalidOption"
			},
			wantErr: "400: InvalidParameter: properties.networkProfile.SoftwareDefinedNetwork: The provided SoftwareDefinedNetwork 'InvalidOption' is invalid.",
		},
	}

	runTests(t, testModeCreate, commontests)
	runTests(t, testModeUpdate, commontests)
	runTests(t, testModeCreate, createtests)
}

func TestOpenShiftClusterStaticValidateMasterProfile(t *testing.T) {
	commonTests := []*validateTest{
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
		{
			name: "disk encryption set is invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.MasterProfile.DiskEncryptionSetID = "invalid"
				oc.Properties.WorkerProfiles[0].DiskEncryptionSetID = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.masterProfile.diskEncryptionSetId: The provided master disk encryption set 'invalid' is invalid.",
		},
		{
			name: "disk encryption set not matching cluster subscriptionId",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.MasterProfile.DiskEncryptionSetID = "/subscriptions/7a3036d1-60a1-4605-8a41-44955e050804/resourceGroups/fakeRG/providers/Microsoft.Compute/diskEncryptionSets/fakeDES1"
			},
			wantErr: "400: InvalidParameter: properties.masterProfile.diskEncryptionSetId: The provided master disk encryption set '/subscriptions/7a3036d1-60a1-4605-8a41-44955e050804/resourceGroups/fakeRG/providers/Microsoft.Compute/diskEncryptionSets/fakeDES1' is invalid: must be in same subscription as cluster.",
		},
		{
			name: "encryption at host invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.MasterProfile.EncryptionAtHost = "Banana"
			},
			wantErr: "400: InvalidParameter: properties.masterProfile.encryptionAtHost: The provided value 'Banana' is invalid.",
		},
		{
			name: "encryption at host empty",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.MasterProfile.EncryptionAtHost = ""
			},
			wantErr: "400: InvalidParameter: properties.masterProfile.encryptionAtHost: The provided value '' is invalid.",
		},
	}

	createTests := []*validateTest{
		{
			name: "disk encryption set is valid",
			modify: func(oc *OpenShiftCluster) {
				desID := fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Compute/diskEncryptionSets/test-disk-encryption-set", subscriptionID)
				oc.Properties.MasterProfile.DiskEncryptionSetID = desID
				oc.Properties.WorkerProfiles[0].DiskEncryptionSetID = desID
			},
		},
	}

	runTests(t, testModeCreate, createTests)
	runTests(t, testModeCreate, commonTests)
	runTests(t, testModeUpdate, commonTests)
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
			name: "vmSize too small (prod)",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].VMSize = "Standard_D2s_v3"
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles['worker'].vmSize: The provided worker VM size 'Standard_D2s_v3' is invalid.",
		},
		{
			name: "vmSize too big (dev)",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].VMSize = "Standard_D4s_v3"
			},
			requireD2sV3Workers: true,
			wantErr:             "400: InvalidParameter: properties.workerProfiles['worker'].vmSize: The provided worker VM size 'Standard_D4s_v3' is invalid.",
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
			wantErr: "400: InvalidParameter: properties.workerProfiles['worker'].subnetId: The provided worker VM subnet '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/different-vnet/subnets/worker' is invalid: must be in the same vnet as master VM subnet '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master'.",
		},
		{
			name: "master and worker subnets not different",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].SubnetID = oc.Properties.MasterProfile.SubnetID
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles['worker'].subnetId: The provided worker VM subnet '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master' is invalid: must be different to master VM subnet '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master'.",
		},
		{
			name: "count too small",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].Count = 1
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles['worker'].count: The provided worker count '1' is invalid.",
		},
		{
			name: "count too big",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].Count = 51
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles['worker'].count: The provided worker count '51' is invalid.",
		},
		{
			name: "disk encryption set not matching master disk encryption set",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.MasterProfile.DiskEncryptionSetID = fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster/providers/Microsoft.Compute/diskEncryptionSets/test-disk-encryption-set", subscriptionID)
				oc.Properties.WorkerProfiles[0].DiskEncryptionSetID = "/subscriptions/7a3036d1-60a1-4605-8a41-44955e050804/resourceGroups/fakeRG/providers/Microsoft.Compute/diskEncryptionSets/fakeDES1"
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles['worker'].subnetId: The provided worker disk encryption set '/subscriptions/7a3036d1-60a1-4605-8a41-44955e050804/resourceGroups/fakeRG/providers/Microsoft.Compute/diskEncryptionSets/fakeDES1' is invalid: must be the same as master disk encryption set '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-cluster/providers/Microsoft.Compute/diskEncryptionSets/test-disk-encryption-set'.",
		},
		{
			name: "encryption at host invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].EncryptionAtHost = "Banana"
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles['worker'].encryptionAtHost: The provided value 'Banana' is invalid.",
		},
		{
			name: "encryption at host empty",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles[0].EncryptionAtHost = ""
			},
			wantErr: "400: InvalidParameter: properties.workerProfiles['worker'].encryptionAtHost: The provided value '' is invalid.",
		},
	}

	// We do not perform this validation on update
	runTests(t, testModeCreate, tests)
}

func TestOpenShiftClusterStaticValidateAPIServerProfile(t *testing.T) {
	commonTests := []*validateTest{
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
			name: "url invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.APIServerProfile.URL = "\x00"
			},
			wantErr: "400: InvalidParameter: properties.apiserverProfile.url: The provided URL '\x00' is invalid.",
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

	createTests := []*validateTest{
		{
			name: "empty url valid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.APIServerProfile.URL = ""
			},
		},
		{
			name: "empty ip valid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.APIServerProfile.IP = ""
			},
		},
	}

	runTests(t, testModeCreate, createTests)
	runTests(t, testModeCreate, commonTests)
	runTests(t, testModeUpdate, commonTests)
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
		{
			name: "empty ip valid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.IngressProfiles[0].IP = ""
			},
		},
	}

	// we don't validate this on update as all fields are immutable and will
	// be validated with "mutable" flag
	runTests(t, testModeCreate, tests)
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
			name:    "pull secret change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ClusterProfile.PullSecret = `{"auths":{}}` },
			wantErr: "400: PropertyChangeNotAllowed: properties.clusterProfile.pullSecret: Changing property 'properties.clusterProfile.pullSecret' is not allowed.",
		},
		{
			name:    "domain change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ClusterProfile.Domain = "invalid" },
			wantErr: "400: PropertyChangeNotAllowed: properties.clusterProfile.domain: Changing property 'properties.clusterProfile.domain' is not allowed.",
		},
		{
			name:    "version change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.ClusterProfile.Version = "4.3.999" },
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
			name: "clientId change",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ServicePrincipalProfile.ClientID = uuid.DefaultGenerator.Generate()
			},
		},
		{
			name:   "clientSecret change",
			modify: func(oc *OpenShiftCluster) { oc.Properties.ServicePrincipalProfile.ClientSecret = "invalid" },
		},
		{
			name: "SoftwareDefinedNetwork should fail to change from OVNKubernetes to OpenShiftSDN",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.SoftwareDefinedNetwork = SoftwareDefinedNetworkOpenShiftSDN
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.networkProfile.softwareDefinedNetwork: Changing property 'properties.networkProfile.softwareDefinedNetwork' is not allowed.",
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
			name:    "worker name change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.WorkerProfiles[0].Name = "new-name" },
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles['new-name'].name: Changing property 'properties.workerProfiles['new-name'].name' is not allowed.",
		},
		{
			name:    "worker vmSize change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.WorkerProfiles[0].VMSize = VMSizeStandardD8sV3 },
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
		{
			name: "number of workerProfiles changes",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles = []WorkerProfile{{}, {}}
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles: Changing property 'properties.workerProfiles' is not allowed.",
		},
		{
			name: "workerProfiles set to nil",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfiles = nil
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.workerProfiles: Changing property 'properties.workerProfiles' is not allowed.",
		},
		{
			name: "systemData set to empty",
			modify: func(oc *OpenShiftCluster) {
				oc.SystemData = &SystemData{}
			},
			wantErr: "400: PropertyChangeNotAllowed: systemData.createdBy: Changing property 'systemData.createdBy' is not allowed.",
		},
		{
			name: "systemData LastUpdated changed",
			modify: func(oc *OpenShiftCluster) {
				oc.SystemData.LastModifiedBy = "Bob"
			},
			wantErr: "400: PropertyChangeNotAllowed: systemData.lastModifiedBy: Changing property 'systemData.lastModifiedBy' is not allowed.",
		},
	}

	runTests(t, testModeUpdate, tests)
}
