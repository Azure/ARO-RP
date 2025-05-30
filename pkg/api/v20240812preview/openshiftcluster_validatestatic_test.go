package v20240812preview

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/test/validate"
	"github.com/Azure/ARO-RP/pkg/api/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/api/util/uuid"
)

type validateTest struct {
	name                string
	clusterName         *string
	location            *string
	current             func(oc *OpenShiftCluster)
	modify              func(oc *OpenShiftCluster)
	requireD2sWorkers   bool
	architectureVersion *api.ArchitectureVersion
	wantErr             string
}

type testMode string

const (
	testModeCreate testMode = "Create"
	testModeUpdate testMode = "Update"

	consoleProfileUrl   = "https://console-openshift-console.apps.cluster.location.aroapp.io/"
	apiserverProfileUrl = "https://api.cluster.location.aroapp.io:6443/"
	apiserverProfileIp  = "1.2.3.4"
	ingressProfileIp    = "1.2.3.4"
)

var (
	subscriptionID    = "00000000-0000-0000-0000-000000000000"
	platformIdentity1 = PlatformWorkloadIdentity{
		ResourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/a-fake-group/providers/Microsoft.RedHatOpenShift/userAssignedIdentities/fake-cluster-name",
	}
	platformIdentity2 = PlatformWorkloadIdentity{
		ResourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/a-fake-group/providers/Microsoft.RedHatOpenShift/userAssignedIdentities/fake-cluster-name-two",
	}
	clusterIdentity1 = UserAssignedIdentity{}
)

func getResourceID(clusterName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourcegroups/resourceGroup/providers/microsoft.redhatopenshift/openshiftclusters/%s", subscriptionID, clusterName)
}

func validSystemData() *SystemData {
	timestamp, err := time.Parse(time.RFC3339, "2021-01-23T12:34:54.0000000Z")
	if err != nil {
		panic(err)
	}

	return &SystemData{
		CreatedBy:          "00000000-0000-0000-0000-000000000000",
		CreatedByType:      CreatedByTypeApplication,
		CreatedAt:          &timestamp,
		LastModifiedBy:     "00000000-0000-0000-0000-000000000000",
		LastModifiedByType: CreatedByTypeApplication,
		LastModifiedAt:     &timestamp,
	}
}

func validOpenShiftCluster(name, location string) *OpenShiftCluster {
	oc := &OpenShiftCluster{
		ID:       getResourceID(name),
		Name:     name,
		Type:     "Microsoft.RedHatOpenShift/OpenShiftClusters",
		Location: location,
		Tags: Tags{
			"key": "value",
		},
		Properties: OpenShiftClusterProperties{
			ProvisioningState: ProvisioningStateSucceeded,
			ClusterProfile: ClusterProfile{
				PullSecret:           `{"auths":{"registry.connect.redhat.com":{"auth":""},"registry.redhat.io":{"auth":""}}}`,
				Domain:               "cluster.location.aroapp.io",
				Version:              "4.10.0",
				ResourceGroupID:      fmt.Sprintf("/subscriptions/%s/resourceGroups/test-cluster", subscriptionID),
				FipsValidatedModules: FipsValidatedModulesDisabled,
			},
			ConsoleProfile: ConsoleProfile{
				URL: "",
			},
			ServicePrincipalProfile: &ServicePrincipalProfile{
				ClientSecret: "clientSecret",
				ClientID:     "11111111-1111-1111-1111-111111111111",
			},
			NetworkProfile: NetworkProfile{
				PodCIDR:      "10.128.0.0/14",
				ServiceCIDR:  "172.30.0.0/16",
				OutboundType: OutboundTypeLoadbalancer,
				LoadBalancerProfile: &LoadBalancerProfile{
					ManagedOutboundIPs: &ManagedOutboundIPs{
						Count: 1,
					},
				},
			},
			MasterProfile: MasterProfile{
				VMSize:           "Standard_D8s_v3",
				EncryptionAtHost: EncryptionAtHostDisabled,
				SubnetID:         fmt.Sprintf("/subscriptions/%s/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/master", subscriptionID),
			},
			WorkerProfiles: []WorkerProfile{
				{
					Name:             "worker",
					VMSize:           "Standard_D4s_v3",
					EncryptionAtHost: EncryptionAtHostDisabled,
					DiskSizeGB:       128,
					SubnetID:         fmt.Sprintf("/subscriptions/%s/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/worker", subscriptionID),
					Count:            3,
				},
			},
			APIServerProfile: APIServerProfile{
				Visibility: VisibilityPublic,
				URL:        "",
				IP:         "",
			},
			IngressProfiles: []IngressProfile{
				{
					Name:       "default",
					Visibility: VisibilityPublic,
					IP:         "",
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
				// default values if not set
				if tt.architectureVersion == nil {
					tt.architectureVersion = pointerutils.ToPtr(api.ArchitectureVersionV2)
				}

				if tt.location == nil {
					tt.location = to.StringPtr("location")
				}

				if tt.clusterName == nil {
					tt.clusterName = to.StringPtr("resourceName")
				}

				v := &openShiftClusterStaticValidator{
					location:          *tt.location,
					domain:            "location.aroapp.io",
					requireD2sWorkers: tt.requireD2sWorkers,
					resourceID:        getResourceID(*tt.clusterName),
					r: azure.Resource{
						SubscriptionID: subscriptionID,
						ResourceGroup:  "resourceGroup",
						Provider:       "Microsoft.RedHatOpenShift",
						ResourceType:   "openshiftClusters",
						ResourceName:   *tt.clusterName,
					},
				}

				validOCForTest := func() *OpenShiftCluster {
					oc := validOpenShiftCluster(*tt.clusterName, *tt.location)
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

					ext := validOCForTest()
					ext.SystemData = validSystemData()
					ext.Properties.ConsoleProfile.URL = consoleProfileUrl
					ext.Properties.APIServerProfile.URL = apiserverProfileUrl
					ext.Properties.APIServerProfile.IP = apiserverProfileIp
					ext.Properties.IngressProfiles[0].IP = ingressProfileIp
					current.Properties.ArchitectureVersion = *tt.architectureVersion

					(&openShiftClusterConverter{}).ToInternal(ext, current)
				}

				err := v.Static(oc, current, v.location, v.domain, tt.requireD2sWorkers, api.ArchitectureVersionV2, v.resourceID)
				if err == nil {
					if tt.wantErr != "" {
						t.Errorf("Expected error %s, got nil", tt.wantErr)
					}
				} else {
					if err.Error() != tt.wantErr {
						t.Errorf("got %s, wanted %s", err, tt.wantErr)
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
	commonTests := []*validateTest{
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

	runTests(t, testModeCreate, commonTests)
	runTests(t, testModeUpdate, commonTests)
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
		{
			name: "workerProfileStatus nonNil",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.WorkerProfilesStatus = []WorkerProfile{
					{
						Name:             "worker",
						VMSize:           "Standard_D4s_v3",
						EncryptionAtHost: EncryptionAtHostDisabled,
						DiskSizeGB:       128,
						SubnetID:         fmt.Sprintf("/subscriptions/%s/resourceGroups/vnet/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/worker", subscriptionID),
						Count:            3,
					},
				}
			},
			wantErr: "400: InvalidParameter: properties.workerProfilesStatus: Worker Profile Status must be set to nil.",
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
		{
			name: "fips validated modules invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterProfile.FipsValidatedModules = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.clusterProfile.fipsValidatedModules: The provided value 'invalid' is invalid.",
		},
		{
			name: "fips validated modules empty",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.ClusterProfile.FipsValidatedModules = ""
			},
			wantErr: "400: InvalidParameter: properties.clusterProfile.fipsValidatedModules: The provided value '' is invalid.",
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
		{
			name: "OutboundType is empty",
			current: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.OutboundType = ""
			},
			wantErr: "",
		},
		{
			name: "OutboundType is invalid",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.OutboundType = "invalid"
			},
			wantErr: "400: InvalidParameter: properties.networkProfile.outboundType: The provided outboundType 'invalid' is invalid: must be UserDefinedRouting or Loadbalancer.",
		},
		{
			name: "OutboundType is invalid with UserDefinedRouting and public ingress",
			current: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.OutboundType = OutboundTypeUserDefinedRouting
				oc.Properties.IngressProfiles[0].Visibility = VisibilityPublic
				oc.Properties.APIServerProfile.Visibility = VisibilityPrivate
			},
			wantErr: "400: InvalidParameter: properties.networkProfile.outboundType: The provided outboundType 'UserDefinedRouting' is invalid: cannot use UserDefinedRouting if either API Server Visibility or Ingress Visibility is public.",
		},
		{
			name: "OutboundType Loadbalancer is valid",
			modify: func(oc *OpenShiftCluster) {
			},
			wantErr: "",
		},
		{
			name: "LoadBalancerProfile invalid when used with UserDefinedRouting",
			current: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.OutboundType = OutboundTypeUserDefinedRouting
				oc.Properties.IngressProfiles[0].Visibility = VisibilityPrivate
				oc.Properties.APIServerProfile.Visibility = VisibilityPrivate
				oc.Properties.NetworkProfile.LoadBalancerProfile = &LoadBalancerProfile{
					ManagedOutboundIPs: &ManagedOutboundIPs{
						Count: 3,
					},
				}
			},
			wantErr: "400: InvalidParameter: properties.networkProfile.loadBalancerProfile: The provided loadBalancerProfile is invalid: cannot use a loadBalancerProfile if outboundType is UserDefinedRouting.",
		},
		{
			name: "Not passing in a LoadBalancerProfile is valid.",
			current: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.OutboundType = OutboundTypeLoadbalancer
				oc.Properties.NetworkProfile.LoadBalancerProfile = nil
			},
			wantErr: "",
		},
		{
			name: "podCidr invalid network",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.PodCIDR = "10.254.0.0/14"
			},
			wantErr: "400: InvalidNetworkAddress: properties.networkProfile.podCidr: The provided pod CIDR '10.254.0.0/14' is invalid, expecting: '10.252.0.0/14'.",
		},
		{
			name: "serviceCidr invalid network",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.ServiceCIDR = "10.0.150.0/16"
			},
			wantErr: "400: InvalidNetworkAddress: properties.networkProfile.serviceCidr: The provided service CIDR '10.0.150.0/16' is invalid, expecting: '10.0.0.0/16'.",
		},
		{
			name: "podCidr invalid CIDR-1",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.PodCIDR = "100.64.0.0/18"
			},
			wantErr: "400: InvalidCIDRRange: properties.networkProfile: Azure Red Hat OpenShift uses 100.64.0.0/16, 169.254.169.0/29, and 100.88.0.0/16 IP address ranges internally. Do not include this '100.64.0.0/18' IP address range in any other CIDR definitions in your cluster.",
		},
		{
			name: "podCidr invalid CIDR-2",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.PodCIDR = "169.254.169.0/29"
			},
			wantErr: "400: InvalidCIDRRange: properties.networkProfile: Azure Red Hat OpenShift uses 100.64.0.0/16, 169.254.169.0/29, and 100.88.0.0/16 IP address ranges internally. Do not include this '169.254.169.0/29' IP address range in any other CIDR definitions in your cluster.",
		},
		{
			name: "podCidr invalid CIDR-3",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.PodCIDR = "100.88.0.0/16"
			},
			wantErr: "400: InvalidCIDRRange: properties.networkProfile: Azure Red Hat OpenShift uses 100.64.0.0/16, 169.254.169.0/29, and 100.88.0.0/16 IP address ranges internally. Do not include this '100.88.0.0/16' IP address range in any other CIDR definitions in your cluster.",
		},
		{
			name: "serviceCidr invalid CIDR-1",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.ServiceCIDR = "100.64.0.0/16"
			},
			wantErr: "400: InvalidCIDRRange: properties.networkProfile: Azure Red Hat OpenShift uses 100.64.0.0/16, 169.254.169.0/29, and 100.88.0.0/16 IP address ranges internally. Do not include this '100.64.0.0/16' IP address range in any other CIDR definitions in your cluster.",
		},
		{
			name: "serviceCidr invalid CIDR-2",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.ServiceCIDR = "169.254.169.1/29"
			},
			wantErr: "400: InvalidCIDRRange: properties.networkProfile: Azure Red Hat OpenShift uses 100.64.0.0/16, 169.254.169.0/29, and 100.88.0.0/16 IP address ranges internally. Do not include this '169.254.169.1/29' IP address range in any other CIDR definitions in your cluster.",
		},
		{
			name: "serviceCidr invalid CIDR-3",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.ServiceCIDR = "100.88.0.0/32"
			},
			wantErr: "400: InvalidCIDRRange: properties.networkProfile: Azure Red Hat OpenShift uses 100.64.0.0/16, 169.254.169.0/29, and 100.88.0.0/16 IP address ranges internally. Do not include this '100.88.0.0/32' IP address range in any other CIDR definitions in your cluster.",
		},
	}

	runTests(t, testModeCreate, tests)
	runTests(t, testModeUpdate, tests)
}

func TestOpenShiftClusterStaticValidateLoadBalancerProfile(t *testing.T) {
	tests := []*validateTest{
		{
			name:    "LoadBalancerProfile is valid",
			wantErr: "",
		},

		{
			name: "LoadBalancerProfile.ManagedOutboundIPs is valid with 20 managed IPs",
			current: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.LoadBalancerProfile = &LoadBalancerProfile{
					ManagedOutboundIPs: &ManagedOutboundIPs{
						Count: 20,
					},
				}
			},
			wantErr: "",
		},
		{
			name: "LoadBalancerProfile.ManagedOutboundIPs is invalid with greater than 20 managed IPs",
			current: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.LoadBalancerProfile = &LoadBalancerProfile{
					ManagedOutboundIPs: &ManagedOutboundIPs{
						Count: 21,
					},
				}
			},
			wantErr: "400: InvalidParameter: properties.networkProfile.loadBalancerProfile.managedOutboundIps.count: The provided managedOutboundIps.count 21 is invalid: managedOutboundIps.count must be in the range of 1 to 20 (inclusive).",
		},
		{
			name: "LoadBalancerProfile.ManagedOutboundIPs is invalid with less than 1 managed IP",
			current: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.LoadBalancerProfile = &LoadBalancerProfile{
					ManagedOutboundIPs: &ManagedOutboundIPs{
						Count: 0,
					},
				}
			},
			wantErr: "400: InvalidParameter: properties.networkProfile.loadBalancerProfile.managedOutboundIps.count: The provided managedOutboundIps.count 0 is invalid: managedOutboundIps.count must be in the range of 1 to 20 (inclusive).",
		},
	}

	createTests := []*validateTest{
		{
			name: "LoadBalancerProfile.EffectiveOutboundIPs is read only",
			current: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.LoadBalancerProfile = &LoadBalancerProfile{
					ManagedOutboundIPs: &ManagedOutboundIPs{
						Count: 1,
					},
					EffectiveOutboundIPs: []EffectiveOutboundIP{
						{
							ID: "someId",
						},
					},
				}
			},
			wantErr: "400: InvalidParameter: properties.networkProfile.loadBalancerProfile.effectiveOutboundIps: The field effectiveOutboundIps is read only.",
		},
	}

	updateOnlyTests := []*validateTest{
		{
			name: "LoadBalancerProfile.ManagedOutboundIPs is invalid with multiple managed IPs and architecture v1",
			current: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.LoadBalancerProfile = &LoadBalancerProfile{
					ManagedOutboundIPs: &ManagedOutboundIPs{
						Count: 20,
					},
				}
			},
			architectureVersion: (*api.ArchitectureVersion)(to.IntPtr(int(api.ArchitectureVersionV1))),
			wantErr:             "400: InvalidParameter: properties.networkProfile.loadBalancerProfile.managedOutboundIps.count: The provided managedOutboundIps.count 20 is invalid: managedOutboundIps.count must be 1, multiple IPs are not supported for this cluster's network architecture.",
		},
	}

	runTests(t, testModeCreate, createTests)
	runTests(t, testModeCreate, tests)
	runTests(t, testModeUpdate, tests)
	runTests(t, testModeUpdate, updateOnlyTests)
}

func TestOpenShiftClusterStaticValidateMasterProfile(t *testing.T) {
	tests := []*validateTest{
		{
			name: "valid",
		},
		{
			name: "vmSize unsupported",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.MasterProfile.VMSize = "Standard_D2s_v3"
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
	runTests(t, testModeCreate, tests)
	runTests(t, testModeUpdate, tests)
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
			requireD2sWorkers: true,
			wantErr:           "400: InvalidParameter: properties.workerProfiles['worker'].vmSize: The provided worker VM size 'Standard_D4s_v3' is invalid.",
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
			name:    "podCidr change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.NetworkProfile.PodCIDR = "10.0.0.0/8" },
			wantErr: "400: PropertyChangeNotAllowed: properties.networkProfile.podCidr: Changing property 'properties.networkProfile.podCidr' is not allowed.",
		},
		{
			name:    "serviceCidr change",
			modify:  func(oc *OpenShiftCluster) { oc.Properties.NetworkProfile.ServiceCIDR = "10.0.0.0/8" },
			wantErr: "400: PropertyChangeNotAllowed: properties.networkProfile.serviceCidr: Changing property 'properties.networkProfile.serviceCidr' is not allowed.",
		},
		{
			name: "outboundType change",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.OutboundType = OutboundTypeUserDefinedRouting
			},
			wantErr: "400: InvalidParameter: properties.networkProfile.outboundType: The provided outboundType 'UserDefinedRouting' is invalid: cannot use UserDefinedRouting if either API Server Visibility or Ingress Visibility is public.",
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
			modify:  func(oc *OpenShiftCluster) { oc.Properties.WorkerProfiles[0].VMSize = "Standard_D8s_v3" },
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
			wantErr: "400: PropertyChangeNotAllowed: systemData: Changing property 'systemData' is not allowed.",
		},
		{
			name: "systemData LastUpdated changed",
			modify: func(oc *OpenShiftCluster) {
				oc.SystemData = &SystemData{}
				oc.SystemData.LastModifiedBy = "Bob"
			},
			wantErr: "400: PropertyChangeNotAllowed: systemData: Changing property 'systemData' is not allowed.",
		},
		{
			name: "update LoadBalancerProfile.ManagedOutboundIPs.Count",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.LoadBalancerProfile = &LoadBalancerProfile{
					ManagedOutboundIPs: &ManagedOutboundIPs{
						Count: 5,
					},
				}
			},
			wantErr: "",
		},
		{
			name: "update LoadBalancerProfile.EffectiveOutboundIPs",
			current: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.LoadBalancerProfile.EffectiveOutboundIPs = []EffectiveOutboundIP{
					{ID: "resourceId"},
				}
			},
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.NetworkProfile.LoadBalancerProfile = &LoadBalancerProfile{
					ManagedOutboundIPs: &ManagedOutboundIPs{
						Count: 5,
					},
					EffectiveOutboundIPs: []EffectiveOutboundIP{
						{
							ID: "BadResourceId",
						},
					},
				}
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.networkProfile.loadBalancerProfile.effectiveOutboundIps: Changing property 'properties.networkProfile.loadBalancerProfile.effectiveOutboundIps' is not allowed.",
		},
	}

	runTests(t, testModeUpdate, tests)
}

func TestOpenShiftClusterStaticValidatePlatformWorkloadIdentityProfile(t *testing.T) {
	validUpgradeableToValue := UpgradeableTo("4.14.29")
	invalidUpgradeableToValue := UpgradeableTo("16.107.invalid")

	createTests := []*validateTest{
		{
			name: "valid empty workloadIdentityProfile",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile = nil
			},
		},
		{
			name: "valid workloadIdentityProfile",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
						"name": platformIdentity1,
					},
				}
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"first": {
							ClientID:    "11111111-1111-1111-1111-111111111111",
							PrincipalID: "SOMETHING",
						},
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
			},
		},
		{
			name: "invalid resourceID",
			modify: func(oc *OpenShiftCluster) {
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"first": {
							ClientID:    "11111111-1111-1111-1111-111111111111",
							PrincipalID: "SOMETHING",
						},
					},
				}
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
						"FAKE-OPERATOR": {
							ResourceID: "BAD",
						},
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
			},
			wantErr: "400: InvalidParameter: properties.platformWorkloadIdentityProfile.PlatformWorkloadIdentities[FAKE-OPERATOR].resourceID: ResourceID BAD formatted incorrectly.",
		},
		{
			name: "wrong resource type",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
						"FAKE-OPERATOR": {
							ResourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/a-fake-group/providers/Microsoft.RedHatOpenShift/otherThing/fake-cluster-name",
						},
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"first": clusterIdentity1,
					},
				}
			},
			wantErr: "400: InvalidParameter: properties.platformWorkloadIdentityProfile.PlatformWorkloadIdentities[FAKE-OPERATOR].resourceID: Resource must be a user assigned identity.",
		},
		{
			name: "no credentials with identities",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
						"name": platformIdentity1,
					},
				}
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"first": clusterIdentity1,
					},
				}
				oc.Properties.ServicePrincipalProfile = &ServicePrincipalProfile{
					ClientID:     "11111111-1111-1111-1111-111111111111",
					ClientSecret: "BAD",
				}
			},
			wantErr: "400: InvalidParameter: properties.servicePrincipalProfile: Cannot use identities and service principal credentials at the same time.",
		},
		{
			name: "cluster identity missing platform workload identity",
			modify: func(oc *OpenShiftCluster) {
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"first": clusterIdentity1,
					},
				}
			},
			wantErr: "400: InvalidParameter: identity: Cluster identity and platform workload identities require each other.",
		},
		{
			name: "platform workload identity missing cluster identity",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
						"operator_name": {},
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
			},
			wantErr: "400: InvalidParameter: identity: Cluster identity and platform workload identities require each other.",
		},
		{
			name: "platform workload identity - cluster identity map is empty",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
						"operator_name": {},
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
				oc.Identity = &ManagedServiceIdentity{}
			},
			wantErr: "400: InvalidParameter: identity: The provided cluster identity is invalid; there should be exactly one.",
		},
		{
			name: "operator name missing",
			modify: func(oc *OpenShiftCluster) {
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"first": clusterIdentity1,
					},
				}
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
						"": {
							ResourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/a-fake-group/providers/Microsoft.RedHatOpenShift/userAssignedIdentities/fake-cluster-name",
						},
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
			},
			wantErr: "400: InvalidParameter: properties.platformWorkloadIdentityProfile.PlatformWorkloadIdentities[].resourceID: Operator name is empty.",
		},
		{
			name: "identity and service principal missing",
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile = nil
				oc.Properties.ServicePrincipalProfile = nil
			},
			wantErr: "400: InvalidParameter: properties.servicePrincipalProfile: Must provide either an identity or service principal credentials.",
		},
		{
			name: "duplicate operator identities",
			modify: func(oc *OpenShiftCluster) {
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"first": clusterIdentity1,
					},
				}
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
						"FAKE-OPERATOR":         platformIdentity1,
						"ANOTHER-FAKE-OPERATOR": platformIdentity1,
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
			},
			wantErr: "400: InvalidParameter: properties.platformWorkloadIdentityProfile.PlatformWorkloadIdentities: ResourceID /subscriptions/12345678-1234-1234-1234-123456789012/resourcegroups/a-fake-group/providers/microsoft.redhatopenshift/userassignedidentities/fake-cluster-name used by multiple identities.",
		},
		{
			name: "duplicate operator identities, different cases",
			modify: func(oc *OpenShiftCluster) {
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"first": clusterIdentity1,
					},
				}
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
						"FAKE-OPERATOR": platformIdentity1,
						"ANOTHER-FAKE-OPERATOR": {
							ResourceID: "/subscriptions/12345678-1234-1234-1234-123456789012/resourceGroups/a-fake-group/providers/Microsoft.RedHatOpenShift/userAssignedIdentities/FAKE-CLUSTER-NAME",
						},
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
			},
			wantErr: "400: InvalidParameter: properties.platformWorkloadIdentityProfile.PlatformWorkloadIdentities: ResourceID /subscriptions/12345678-1234-1234-1234-123456789012/resourcegroups/a-fake-group/providers/microsoft.redhatopenshift/userassignedidentities/fake-cluster-name used by multiple identities.",
		},
		{
			name: "valid UpgradeableTo value",
			modify: func(oc *OpenShiftCluster) {
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"Dummy": {},
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
						"FAKE-OPERATOR": platformIdentity1,
					},
					UpgradeableTo: &validUpgradeableToValue,
				}
			},
		},
		{
			name: "invalid UpgradeableTo value",
			modify: func(oc *OpenShiftCluster) {
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"Dummy": {},
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
						"FAKE-OPERATOR": platformIdentity1,
					},
					UpgradeableTo: &invalidUpgradeableToValue,
				}
			},
			wantErr: `400: InvalidParameter: properties.platformWorkloadIdentityProfile.UpgradeableTo[16.107.invalid]: UpgradeableTo must be a valid OpenShift version in the format 'x.y.z'.`,
		},
		{
			name: "No platform identities provided in PlatformWorkloadIdentityProfile - nil",
			modify: func(oc *OpenShiftCluster) {
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"Dummy": {},
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					UpgradeableTo: &invalidUpgradeableToValue,
				}
			},
			wantErr: "400: InvalidParameter: properties.platformWorkloadIdentityProfile.platformWorkloadIdentities: The set of platform workload identities cannot be empty.",
		},
		{
			name: "No platform identities provided in PlatformWorkloadIdentityProfile - empty map",
			modify: func(oc *OpenShiftCluster) {
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"Dummy": {},
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{},
					UpgradeableTo:              &invalidUpgradeableToValue,
				}
			},
			wantErr: "400: InvalidParameter: properties.platformWorkloadIdentityProfile.platformWorkloadIdentities: The set of platform workload identities cannot be empty.",
		},
	}

	updateTests := []*validateTest{
		{
			name: "addition of operator identity",
			current: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
						"FAKE-OPERATOR": platformIdentity1,
					},
				}
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"first": clusterIdentity1,
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
			},
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["ANOTHER-FAKE-OPERATOR"] = platformIdentity2
			},
		},
		{
			name: "invalid change of operator identity name",
			current: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
						"FAKE-OPERATOR": platformIdentity1,
					},
				}
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"first": clusterIdentity1,
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
			},
			modify: func(oc *OpenShiftCluster) {
				pwi := map[string]PlatformWorkloadIdentity{
					"FAKE-OPERATOR-OTHER": platformIdentity1,
				}
				oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities = pwi
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.platformWorkloadIdentityProfile.platformWorkloadIdentities: Operator identity cannot be removed or have its name changed.",
		},
		{
			name: "valid change of operator identity resource ID",
			current: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
						"FAKE-OPERATOR": platformIdentity1,
					},
				}
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"first": clusterIdentity1,
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
			},
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities["FAKE-OPERATOR"] = platformIdentity2
			},
		},
		{
			name: "change of operator identity order",
			current: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
						"OPERATOR-1": platformIdentity1,
						"OPERATOR-2": platformIdentity2,
					},
				}
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"first": clusterIdentity1,
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
			},
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities = map[string]PlatformWorkloadIdentity{
					"OPERATOR-1": platformIdentity1,
					"OPERATOR-2": platformIdentity2,
				}
			},
		},
		{
			name: "invalid change of operator identity name and resource ID",
			current: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
						"FAKE-OPERATOR": platformIdentity1,
					},
				}
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"first": clusterIdentity1,
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
			},
			modify: func(oc *OpenShiftCluster) {
				pwi := map[string]PlatformWorkloadIdentity{
					"FAKE-OPERATOR-OTHER": platformIdentity2,
				}
				oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities = pwi
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.platformWorkloadIdentityProfile.platformWorkloadIdentities: Operator identity cannot be removed or have its name changed.",
		},
		{
			name: "invalid removal of identity",
			current: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
						"operator1": platformIdentity1,
						"operator2": platformIdentity2,
					},
				}
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"first": clusterIdentity1,
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
			},
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities = map[string]PlatformWorkloadIdentity{
					"operator1": platformIdentity1,
				}
			},
			wantErr: "400: PropertyChangeNotAllowed: properties.platformWorkloadIdentityProfile.platformWorkloadIdentities: Operator identity cannot be removed or have its name changed.",
		},
		{
			name: "No platform identities provided in PlatformWorkloadIdentityProfile - empty map",
			current: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
						"operator1": platformIdentity1,
					},
				}
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"first": clusterIdentity1,
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
			},
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities = map[string]PlatformWorkloadIdentity{}
			},
			wantErr: "400: InvalidParameter: properties.platformWorkloadIdentityProfile.platformWorkloadIdentities: The set of platform workload identities cannot be empty.",
		},
		{
			name: "No platform identities provided in PlatformWorkloadIdentityProfile - nil",
			current: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile = &PlatformWorkloadIdentityProfile{
					PlatformWorkloadIdentities: map[string]PlatformWorkloadIdentity{
						"operator1": platformIdentity1,
					},
				}
				oc.Identity = &ManagedServiceIdentity{
					UserAssignedIdentities: map[string]UserAssignedIdentity{
						"first": clusterIdentity1,
					},
				}
				oc.Properties.ServicePrincipalProfile = nil
			},
			modify: func(oc *OpenShiftCluster) {
				oc.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities = nil
			},
			wantErr: "400: InvalidParameter: properties.platformWorkloadIdentityProfile.platformWorkloadIdentities: The set of platform workload identities cannot be empty.",
		},
	}

	runTests(t, testModeCreate, createTests)
	runTests(t, testModeUpdate, updateTests)
}
