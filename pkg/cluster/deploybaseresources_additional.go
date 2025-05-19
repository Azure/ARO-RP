package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"strings"

	sdknetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v2"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-09-01/storage"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/platformworkloadidentity"
	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) denyAssignment() *arm.Resource {
	excludePrincipals := []mgmtauthorization.Principal{}
	if m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		for _, identity := range m.doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
			excludePrincipals = append(excludePrincipals, mgmtauthorization.Principal{
				ID:   pointerutils.ToPtr(identity.ObjectID),
				Type: to.StringPtr(string(mgmtauthorization.ServicePrincipal)),
			})
		}
		excludePrincipals = append(excludePrincipals, mgmtauthorization.Principal{
			ID:   pointerutils.ToPtr(m.fpServicePrincipalID),
			Type: pointerutils.ToPtr(string(mgmtauthorization.ServicePrincipal)),
		})
	} else {
		excludePrincipals = append(excludePrincipals, mgmtauthorization.Principal{
			ID:   &m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.SPObjectID,
			Type: to.StringPtr(string(mgmtauthorization.ServicePrincipal)),
		})
	}

	resource := &arm.Resource{
		Resource: &mgmtauthorization.DenyAssignment{
			Name: to.StringPtr("[guid(resourceGroup().id, 'ARO cluster resource group deny assignment')]"),
			Type: to.StringPtr("Microsoft.Authorization/denyAssignments"),
			DenyAssignmentProperties: &mgmtauthorization.DenyAssignmentProperties{
				DenyAssignmentName: to.StringPtr("[guid(resourceGroup().id, 'ARO cluster resource group deny assignment')]"),
				Permissions: &[]mgmtauthorization.DenyAssignmentPermission{
					{
						Actions: &[]string{
							"*/action",
							"*/delete",
							"*/write",
						},
						NotActions: &[]string{
							"Microsoft.Compute/disks/beginGetAccess/action",
							"Microsoft.Compute/disks/endGetAccess/action",
							"Microsoft.Compute/disks/write",
							"Microsoft.Insights/ActionGroups/write",
							"Microsoft.Insights/ActionGroups/delete",
							"Microsoft.Insights/MetricAlerts/write",
							"Microsoft.Insights/MetricAlerts/delete",
							"Microsoft.Insights/ActivityLogAlerts/write",
							"Microsoft.Insights/ActivityLogAlerts/delete",
							"Microsoft.Compute/snapshots/beginGetAccess/action",
							"Microsoft.Compute/snapshots/delete",
							"Microsoft.Compute/snapshots/endGetAccess/action",
							"Microsoft.Compute/snapshots/write",
							"Microsoft.Network/networkInterfaces/effectiveRouteTable/action",
							"Microsoft.Network/networkSecurityGroups/join/action",
							"Microsoft.Resources/tags/*", // Enable tagging for Resources RP only
							"Microsoft.PolicyInsights/remediations/write",
							"Microsoft.PolicyInsights/remediations/delete",
						},
					},
				},
				Scope: &m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID,
				Principals: &[]mgmtauthorization.Principal{
					{
						ID:   to.StringPtr("00000000-0000-0000-0000-000000000000"),
						Type: to.StringPtr("SystemDefined"),
					},
				},
				ExcludePrincipals: &excludePrincipals,
				IsSystemProtected: to.BoolPtr(true),
			},
		},
		APIVersion: azureclient.APIVersion("Microsoft.Authorization/denyAssignments"),
	}
	return resource
}

func (m *manager) clusterServicePrincipalRBAC() *arm.Resource {
	return rbac.ResourceGroupRoleAssignmentWithName(
		rbac.RoleContributor,
		"'"+m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.SPObjectID+"'",
		"guid(resourceGroup().id, 'SP / Contributor')",
	)
}

func (m *manager) platformWorkloadIdentityRBAC() ([]*arm.Resource, error) {
	if !m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		return nil, nil
	}

	resources := []*arm.Resource{}
	platformWIRolesByRoleName := m.platformWorkloadIdentityRolesByVersion.GetPlatformWorkloadIdentityRolesByRoleName()
	platformWorkloadIdentities := m.doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities

	for name, identity := range platformWorkloadIdentities {
		role, exists := platformWIRolesByRoleName[name]
		if !exists {
			return nil, platformworkloadidentity.GetPlatformWorkloadIdentityMismatchError(m.doc.OpenShiftCluster, platformWIRolesByRoleName)
		}

		if strings.TrimSpace(identity.ObjectID) == "" {
			return nil, fmt.Errorf("WI object ID '%s' is invalid for WI with resource ID %s", identity.ObjectID, identity.ResourceID)
		}

		roleID := stringutils.LastTokenByte(role.RoleDefinitionID, '/')
		resources = append(resources, m.workloadIdentityResourceGroupRBAC(roleID, identity.ObjectID))
	}
	return resources, nil
}

func (m *manager) workloadIdentityResourceGroupRBAC(roleID, objID string) *arm.Resource {
	if !m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		return nil
	}

	r := rbac.ResourceGroupRoleAssignmentWithName(
		roleID,
		"'"+objID+"'",
		"guid(resourceGroup().id, '"+roleID+"')",
	)
	return r
}

func (m *manager) fpspStorageBlobContributorRBAC(storageAccountName, principalID string) (*arm.Resource, error) {
	if !m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		return nil, fmt.Errorf("fpspStorageBlobContributorRBAC called for a Cluster Service Principal cluster")
	}
	resourceTypeStorageAccount := "Microsoft.Storage/storageAccounts"
	return rbac.ResourceRoleAssignmentWithName(
		rbac.RoleStorageBlobDataContributor,
		fmt.Sprintf("'%s'", principalID),
		resourceTypeStorageAccount,
		fmt.Sprintf("'%s'", storageAccountName),
		fmt.Sprintf("concat('%s', '/Microsoft.Authorization/', guid(resourceId('%s', '%s')))", storageAccountName, resourceTypeStorageAccount, storageAccountName),
	), nil
}

// storageAccount will return storage account resource.
// Legacy storage accounts (public) are not encrypted and cannot be retrofitted.
// The flag controls this behavior in update/create.
func (m *manager) storageAccount(name, region string, ocpSubnets []string, encrypted bool, setSasPolicy bool) *arm.Resource {
	virtualNetworkRules := []mgmtstorage.VirtualNetworkRule{
		{
			VirtualNetworkResourceID: to.StringPtr("/subscriptions/" + m.env.SubscriptionID() + "/resourceGroups/" + m.env.ResourceGroup() + "/providers/Microsoft.Network/virtualNetworks/rp-pe-vnet-001/subnets/rp-pe-subnet"),
			Action:                   mgmtstorage.ActionAllow,
		},
		{
			VirtualNetworkResourceID: to.StringPtr("/subscriptions/" + m.env.SubscriptionID() + "/resourceGroups/" + m.env.ResourceGroup() + "/providers/Microsoft.Network/virtualNetworks/rp-vnet/subnets/rp-subnet"),
			Action:                   mgmtstorage.ActionAllow,
		},
	}

	// add OCP subnets which have Microsoft.Storage service endpoint enabled
	for _, subnet := range ocpSubnets {
		virtualNetworkRules = append(virtualNetworkRules, mgmtstorage.VirtualNetworkRule{
			VirtualNetworkResourceID: to.StringPtr(subnet),
			Action:                   mgmtstorage.ActionAllow,
		})
	}

	// when installing via Hive we need to allow Hive to persist the installConfig graph in the cluster's storage account
	// TODO: add AKS shard support
	hiveShard := 1
	if m.installViaHive && strings.Index(name, "cluster") == 0 {
		virtualNetworkRules = append(virtualNetworkRules, mgmtstorage.VirtualNetworkRule{
			VirtualNetworkResourceID: to.StringPtr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/aks-net/subnets/PodSubnet-%03d", m.env.SubscriptionID(), m.env.ResourceGroup(), hiveShard)),
			Action:                   mgmtstorage.ActionAllow,
		})
	}

	// Prod includes a gateway rule as well
	// Once we reach a PLS limit (1000) within a vnet , we may need some refactoring here
	// https://docs.microsoft.com/en-us/azure/azure-resource-manager/management/azure-subscription-service-limits#private-link-limits
	if !m.env.IsLocalDevelopmentMode() {
		virtualNetworkRules = append(virtualNetworkRules, mgmtstorage.VirtualNetworkRule{
			VirtualNetworkResourceID: to.StringPtr("/subscriptions/" + m.env.SubscriptionID() + "/resourceGroups/" + m.env.GatewayResourceGroup() + "/providers/Microsoft.Network/virtualNetworks/gateway-vnet/subnets/gateway-subnet"),
			Action:                   mgmtstorage.ActionAllow,
		})
	}

	sa := &mgmtstorage.Account{
		Kind: mgmtstorage.KindStorageV2,
		Sku: &mgmtstorage.Sku{
			Name: "Standard_LRS",
		},
		AccountProperties: &mgmtstorage.AccountProperties{
			AllowBlobPublicAccess:  to.BoolPtr(false),
			EnableHTTPSTrafficOnly: to.BoolPtr(true),
			MinimumTLSVersion:      mgmtstorage.MinimumTLSVersionTLS12,
			NetworkRuleSet: &mgmtstorage.NetworkRuleSet{
				Bypass:              mgmtstorage.BypassAzureServices,
				VirtualNetworkRules: &virtualNetworkRules,
				DefaultAction:       "Deny",
			},
		},
		Name:     &name,
		Location: &region,
		Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
	}

	// For Workload Identity Cluster disable shared access keys, only User Delegated SAS are allowed
	if m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		sa.AllowSharedKeyAccess = to.BoolPtr(false)
		if setSasPolicy {
			sa.SasPolicy = &mgmtstorage.SasPolicy{
				SasExpirationPeriod: to.StringPtr("0.01:00:00"),
				ExpirationAction:    to.StringPtr("Log"),
			}
		}
	}

	// In development API calls originates from user laptop so we allow all.
	// TODO: Move to development on VPN so we can make this IPRule.  Will be done as part of Simply secure v2 work
	if m.env.IsLocalDevelopmentMode() {
		sa.NetworkRuleSet.DefaultAction = mgmtstorage.DefaultActionAllow
	}
	// When migrating storage accounts for old clusters we are not able to change
	// encryption which is why we have this encryption flag. We will not add this
	// retrospectively to old clusters
	// If a storage account already has encryption enabled and the encrypted
	// bool is set to false, it will still maintain the encryption on the storage account.
	if encrypted {
		sa.AccountProperties.Encryption = &mgmtstorage.Encryption{
			RequireInfrastructureEncryption: to.BoolPtr(true),
			Services: &mgmtstorage.EncryptionServices{
				Blob: &mgmtstorage.EncryptionService{
					KeyType: mgmtstorage.KeyTypeAccount,
					Enabled: to.BoolPtr(true),
				},
				File: &mgmtstorage.EncryptionService{
					KeyType: mgmtstorage.KeyTypeAccount,
					Enabled: to.BoolPtr(true),
				},
				Table: &mgmtstorage.EncryptionService{
					KeyType: mgmtstorage.KeyTypeAccount,
					Enabled: to.BoolPtr(true),
				},
				Queue: &mgmtstorage.EncryptionService{
					KeyType: mgmtstorage.KeyTypeAccount,
					Enabled: to.BoolPtr(true),
				},
			},
			KeySource: mgmtstorage.KeySourceMicrosoftStorage,
		}
	}

	return &arm.Resource{
		Resource:   sa,
		APIVersion: azureclient.APIVersion("Microsoft.Storage"),
	}
}

func (m *manager) storageAccountBlobContainer(storageAccountName, name string) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtstorage.BlobContainer{
			Name: to.StringPtr(storageAccountName + "/default/" + name),
			Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Storage"),
		DependsOn: []string{
			"Microsoft.Storage/storageAccounts/" + storageAccountName,
		},
	}
}

func (m *manager) networkPrivateLinkService(azureRegion string) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.PrivateLinkService{
			PrivateLinkServiceProperties: &mgmtnetwork.PrivateLinkServiceProperties{
				LoadBalancerFrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
					{
						ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', '%s-internal', 'internal-lb-ip-v4')]", m.doc.OpenShiftCluster.Properties.InfraID)),
					},
				},
				IPConfigurations: &[]mgmtnetwork.PrivateLinkServiceIPConfiguration{
					{
						PrivateLinkServiceIPConfigurationProperties: &mgmtnetwork.PrivateLinkServiceIPConfigurationProperties{
							Subnet: &mgmtnetwork.Subnet{
								ID: &m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID,
							},
						},
						Name: to.StringPtr(m.doc.OpenShiftCluster.Properties.InfraID + "-pls-nic"),
					},
				},
				Visibility: &mgmtnetwork.PrivateLinkServicePropertiesVisibility{
					Subscriptions: &[]string{
						m.env.SubscriptionID(),
					},
				},
				AutoApproval: &mgmtnetwork.PrivateLinkServicePropertiesAutoApproval{
					Subscriptions: &[]string{
						m.env.SubscriptionID(),
					},
				},
			},
			Name:     to.StringPtr(m.doc.OpenShiftCluster.Properties.InfraID + "-pls"),
			Type:     to.StringPtr("Microsoft.Network/privateLinkServices"),
			Location: &azureRegion,
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
		DependsOn: []string{
			"Microsoft.Network/loadBalancers/" + m.doc.OpenShiftCluster.Properties.InfraID + "-internal",
		},
	}
}

func (m *manager) networkPrivateEndpoint() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.PrivateEndpoint{
			PrivateEndpointProperties: &mgmtnetwork.PrivateEndpointProperties{
				Subnet: &mgmtnetwork.Subnet{
					ID: to.StringPtr(m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID),
				},
				ManualPrivateLinkServiceConnections: &[]mgmtnetwork.PrivateLinkServiceConnection{
					{
						Name: to.StringPtr("gateway-plsconnection"),
						PrivateLinkServiceConnectionProperties: &mgmtnetwork.PrivateLinkServiceConnectionProperties{
							// TODO: in the future we will need multiple PLSes.
							// It will be necessary to decide which the PLS for
							// a cluster somewhere around here.
							PrivateLinkServiceID: to.StringPtr("/subscriptions/" + m.env.SubscriptionID() + "/resourceGroups/" + m.env.GatewayResourceGroup() + "/providers/Microsoft.Network/privateLinkServices/gateway-pls-001"),
						},
					},
				},
			},
			Name:     to.StringPtr(m.doc.OpenShiftCluster.Properties.InfraID + "-pe"),
			Type:     to.StringPtr("Microsoft.Network/privateEndpoints"),
			Location: &m.doc.OpenShiftCluster.Location,
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (m *manager) networkPublicIPAddress(azureRegion string, name string) *arm.Resource {
	zones := to.StringSlicePtr([]string{})

	if m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.OutboundIPAvailabilityZones != nil {
		zones = to.StringSlicePtr(m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.OutboundIPAvailabilityZones)
	}

	return &arm.Resource{
		Resource: &mgmtnetwork.PublicIPAddress{
			Sku: &mgmtnetwork.PublicIPAddressSku{
				Name: mgmtnetwork.PublicIPAddressSkuNameStandard,
			},
			PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
				PublicIPAllocationMethod: mgmtnetwork.Static,
			},
			Zones:    zones,
			Name:     &name,
			Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
			Location: &azureRegion,
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (m *manager) networkInternalLoadBalancer(azureRegion string) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.LoadBalancer{
			Sku: &mgmtnetwork.LoadBalancerSku{
				Name: mgmtnetwork.LoadBalancerSkuNameStandard,
			},
			LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
				FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
					{
						FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
							PrivateIPAllocationMethod: mgmtnetwork.Dynamic,
							Subnet: &mgmtnetwork.Subnet{
								ID: to.StringPtr(m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID),
							},
						},
						Name: to.StringPtr("internal-lb-ip-v4"),
					},
				},
				BackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
					{
						Name: &m.doc.OpenShiftCluster.Properties.InfraID,
					},
					{
						Name: to.StringPtr("ssh-0"),
					},
					{
						Name: to.StringPtr("ssh-1"),
					},
					{
						Name: to.StringPtr("ssh-2"),
					},
				},
				LoadBalancingRules: &[]mgmtnetwork.LoadBalancingRule{
					{
						LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', '%s-internal', 'internal-lb-ip-v4')]", m.doc.OpenShiftCluster.Properties.InfraID)),
							},
							BackendAddressPool: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s-internal', '%[1]s')]", m.doc.OpenShiftCluster.Properties.InfraID)),
							},
							Probe: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/probes', '%s-internal', 'api-internal-probe')]", m.doc.OpenShiftCluster.Properties.InfraID)),
							},
							Protocol:             mgmtnetwork.TransportProtocolTCP,
							LoadDistribution:     mgmtnetwork.LoadDistributionDefault,
							FrontendPort:         to.Int32Ptr(6443),
							BackendPort:          to.Int32Ptr(6443),
							IdleTimeoutInMinutes: to.Int32Ptr(30),
							DisableOutboundSnat:  to.BoolPtr(true),
						},
						Name: to.StringPtr("api-internal-v4"),
					},
					{
						LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', '%s-internal', 'internal-lb-ip-v4')]", m.doc.OpenShiftCluster.Properties.InfraID)),
							},
							BackendAddressPool: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s-internal', '%[1]s')]", m.doc.OpenShiftCluster.Properties.InfraID)),
							},
							Probe: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/probes', '%s-internal', 'sint-probe')]", m.doc.OpenShiftCluster.Properties.InfraID)),
							},
							Protocol:             mgmtnetwork.TransportProtocolTCP,
							LoadDistribution:     mgmtnetwork.LoadDistributionDefault,
							FrontendPort:         to.Int32Ptr(22623),
							BackendPort:          to.Int32Ptr(22623),
							IdleTimeoutInMinutes: to.Int32Ptr(30),
						},
						Name: to.StringPtr("sint-v4"),
					},
					{
						LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', '%s-internal', 'internal-lb-ip-v4')]", m.doc.OpenShiftCluster.Properties.InfraID)),
							},
							BackendAddressPool: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s-internal', 'ssh-0')]", m.doc.OpenShiftCluster.Properties.InfraID)),
							},
							Probe: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/probes', '%s-internal', 'ssh')]", m.doc.OpenShiftCluster.Properties.InfraID)),
							},
							Protocol:             mgmtnetwork.TransportProtocolTCP,
							LoadDistribution:     mgmtnetwork.LoadDistributionDefault,
							FrontendPort:         to.Int32Ptr(2200),
							BackendPort:          to.Int32Ptr(22),
							IdleTimeoutInMinutes: to.Int32Ptr(30),
							DisableOutboundSnat:  to.BoolPtr(true),
						},
						Name: to.StringPtr("ssh-0"),
					},
					{
						LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', '%s-internal', 'internal-lb-ip-v4')]", m.doc.OpenShiftCluster.Properties.InfraID)),
							},
							BackendAddressPool: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s-internal', 'ssh-1')]", m.doc.OpenShiftCluster.Properties.InfraID)),
							},
							Probe: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/probes', '%s-internal', 'ssh')]", m.doc.OpenShiftCluster.Properties.InfraID)),
							},
							Protocol:             mgmtnetwork.TransportProtocolTCP,
							LoadDistribution:     mgmtnetwork.LoadDistributionDefault,
							FrontendPort:         to.Int32Ptr(2201),
							BackendPort:          to.Int32Ptr(22),
							IdleTimeoutInMinutes: to.Int32Ptr(30),
							DisableOutboundSnat:  to.BoolPtr(true),
						},
						Name: to.StringPtr("ssh-1"),
					},
					{
						LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', '%s-internal', 'internal-lb-ip-v4')]", m.doc.OpenShiftCluster.Properties.InfraID)),
							},
							BackendAddressPool: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s-internal', 'ssh-2')]", m.doc.OpenShiftCluster.Properties.InfraID)),
							},
							Probe: &mgmtnetwork.SubResource{
								ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/probes', '%s-internal', 'ssh')]", m.doc.OpenShiftCluster.Properties.InfraID)),
							},
							Protocol:             mgmtnetwork.TransportProtocolTCP,
							LoadDistribution:     mgmtnetwork.LoadDistributionDefault,
							FrontendPort:         to.Int32Ptr(2202),
							BackendPort:          to.Int32Ptr(22),
							IdleTimeoutInMinutes: to.Int32Ptr(30),
							DisableOutboundSnat:  to.BoolPtr(true),
						},
						Name: to.StringPtr("ssh-2"),
					},
				},
				Probes: &[]mgmtnetwork.Probe{
					{
						ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
							Protocol:          mgmtnetwork.ProbeProtocolHTTPS,
							Port:              to.Int32Ptr(6443),
							IntervalInSeconds: to.Int32Ptr(5),
							NumberOfProbes:    to.Int32Ptr(2),
							RequestPath:       to.StringPtr("/readyz"),
						},
						Name: to.StringPtr("api-internal-probe"),
					},
					{
						ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
							Protocol:          mgmtnetwork.ProbeProtocolHTTPS,
							Port:              to.Int32Ptr(22623),
							IntervalInSeconds: to.Int32Ptr(5),
							NumberOfProbes:    to.Int32Ptr(2),
							RequestPath:       to.StringPtr("/healthz"),
						},
						Name: to.StringPtr("sint-probe"),
					},
					{
						ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
							Protocol:          mgmtnetwork.ProbeProtocolTCP,
							Port:              to.Int32Ptr(22),
							IntervalInSeconds: to.Int32Ptr(5),
							NumberOfProbes:    to.Int32Ptr(2),
						},
						Name: to.StringPtr("ssh"),
					},
				},
			},
			Name:     to.StringPtr(m.doc.OpenShiftCluster.Properties.InfraID + "-internal"),
			Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
			Location: &azureRegion,
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (m *manager) networkPublicLoadBalancer(azureRegion string, outboundIPs []api.ResourceReference) *arm.Resource {
	lb := &sdknetwork.LoadBalancer{
		SKU: &sdknetwork.LoadBalancerSKU{
			Name: pointerutils.ToPtr(sdknetwork.LoadBalancerSKUNameStandard),
		},
		Properties: &sdknetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: []*sdknetwork.FrontendIPConfiguration{},
			BackendAddressPools: []*sdknetwork.BackendAddressPool{
				{
					Name: pointerutils.ToPtr(m.doc.OpenShiftCluster.Properties.InfraID),
				},
			},
			LoadBalancingRules: []*sdknetwork.LoadBalancingRule{}, //required to override default LB rules for port 80 and 443
			Probes:             []*sdknetwork.Probe{},             //required to override default LB rules for port 80 and 443
			OutboundRules: []*sdknetwork.OutboundRule{
				{
					Properties: &sdknetwork.OutboundRulePropertiesFormat{
						FrontendIPConfigurations: []*sdknetwork.SubResource{},
						BackendAddressPool: &sdknetwork.SubResource{
							ID: pointerutils.ToPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s', '%[1]s')]", m.doc.OpenShiftCluster.Properties.InfraID)),
						},
						Protocol:             pointerutils.ToPtr(sdknetwork.LoadBalancerOutboundRuleProtocolAll),
						IdleTimeoutInMinutes: pointerutils.ToPtr(int32(30)),
					},
					Name: pointerutils.ToPtr("outbound-rule-v4"),
				},
			},
		},
		Name:     &m.doc.OpenShiftCluster.Properties.InfraID,
		Type:     pointerutils.ToPtr("Microsoft.Network/loadBalancers"),
		Location: &azureRegion,
	}

	if m.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
		lb.Properties.FrontendIPConfigurations = append(lb.Properties.FrontendIPConfigurations, &sdknetwork.FrontendIPConfiguration{
			Properties: &sdknetwork.FrontendIPConfigurationPropertiesFormat{
				PublicIPAddress: &sdknetwork.PublicIPAddress{
					ID: pointerutils.ToPtr("[resourceId('Microsoft.Network/publicIPAddresses', '" + m.doc.OpenShiftCluster.Properties.InfraID + "-pip-v4')]"),
				},
			},
			Name: pointerutils.ToPtr("public-lb-ip-v4"),
		})

		lb.Properties.LoadBalancingRules = append(lb.Properties.LoadBalancingRules, &sdknetwork.LoadBalancingRule{
			Properties: &sdknetwork.LoadBalancingRulePropertiesFormat{
				FrontendIPConfiguration: &sdknetwork.SubResource{
					ID: pointerutils.ToPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', '%s', 'public-lb-ip-v4')]", m.doc.OpenShiftCluster.Properties.InfraID)),
				},
				BackendAddressPool: &sdknetwork.SubResource{
					ID: pointerutils.ToPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', '%s', '%[1]s')]", m.doc.OpenShiftCluster.Properties.InfraID)),
				},
				Probe: &sdknetwork.SubResource{
					ID: pointerutils.ToPtr(fmt.Sprintf("[resourceId('Microsoft.Network/loadBalancers/probes', '%s', 'api-internal-probe')]", m.doc.OpenShiftCluster.Properties.InfraID)),
				},
				Protocol:             pointerutils.ToPtr(sdknetwork.TransportProtocolTCP),
				LoadDistribution:     pointerutils.ToPtr(sdknetwork.LoadDistributionDefault),
				FrontendPort:         pointerutils.ToPtr(int32(6443)),
				BackendPort:          pointerutils.ToPtr(int32(6443)),
				IdleTimeoutInMinutes: pointerutils.ToPtr(int32(30)),
				DisableOutboundSnat:  pointerutils.ToPtr(true),
			},
			Name: pointerutils.ToPtr("api-internal-v4"),
		})

		lb.Properties.Probes = append(lb.Properties.Probes, &sdknetwork.Probe{
			Properties: &sdknetwork.ProbePropertiesFormat{
				Protocol:          pointerutils.ToPtr(sdknetwork.ProbeProtocolHTTPS),
				Port:              pointerutils.ToPtr(int32(6443)),
				IntervalInSeconds: pointerutils.ToPtr(int32(5)),
				NumberOfProbes:    pointerutils.ToPtr(int32(2)),
				RequestPath:       pointerutils.ToPtr("/readyz"),
			},
			Name: pointerutils.ToPtr("api-internal-probe"),
		})
	}

	if m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs != nil {
		for i := len(lb.Properties.FrontendIPConfigurations); i < m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs.Count; i++ {
			resourceGroupID := m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID
			frontendIPConfigName := stringutils.LastTokenByte(outboundIPs[i].ID, '/')
			frontendConfigID := fmt.Sprintf("%s/providers/Microsoft.Network/loadBalancers/%s/frontendIPConfigurations/%s", resourceGroupID, *lb.Name, frontendIPConfigName)
			lb.Properties.FrontendIPConfigurations = append(lb.Properties.FrontendIPConfigurations, newFrontendIPConfig(frontendIPConfigName, frontendConfigID, outboundIPs[i].ID))
		}

		for i := 0; i < m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs.Count; i++ {
			resourceGroupID := m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID
			if i == 0 && m.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
				frontendIPConfigName := "public-lb-ip-v4"
				frontendConfigID := fmt.Sprintf("%s/providers/Microsoft.Network/loadBalancers/%s/frontendIPConfigurations/%s", resourceGroupID, *lb.Name, frontendIPConfigName)
				lb.Properties.OutboundRules[0].Properties.FrontendIPConfigurations = append(lb.Properties.OutboundRules[0].Properties.FrontendIPConfigurations, newOutboundRuleFrontendIPConfig(frontendConfigID))
				continue
			}
			frontendIPConfigName := stringutils.LastTokenByte(outboundIPs[i].ID, '/')
			frontendConfigID := fmt.Sprintf("%s/providers/Microsoft.Network/loadBalancers/%s/frontendIPConfigurations/%s", resourceGroupID, *lb.Name, frontendIPConfigName)
			lb.Properties.OutboundRules[0].Properties.FrontendIPConfigurations = append(lb.Properties.OutboundRules[0].Properties.FrontendIPConfigurations, newOutboundRuleFrontendIPConfig(frontendConfigID))
		}
	}

	armResource := &arm.Resource{
		Resource:   lb,
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
		DependsOn:  []string{},
	}

	if m.doc.OpenShiftCluster.Properties.NetworkProfile.LoadBalancerProfile.ManagedOutboundIPs == nil && m.doc.OpenShiftCluster.Properties.APIServerProfile.Visibility == api.VisibilityPublic {
		armResource.DependsOn = append(armResource.DependsOn, "Microsoft.Network/publicIPAddresses/"+m.doc.OpenShiftCluster.Properties.InfraID+"-pip-v4")
	}

	for _, ip := range outboundIPs {
		ipName := stringutils.LastTokenByte(ip.ID, '/')
		armResource.DependsOn = append(armResource.DependsOn, "Microsoft.Network/publicIPAddresses/"+ipName)
	}

	return armResource
}
