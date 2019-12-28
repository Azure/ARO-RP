package deploy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2019-08-01/documentdb"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"

	"github.com/Azure/ARO-RP/pkg/util/arm"
)

var apiVersions = map[string]string{
	"authorization": "2018-09-01-preview",
	"compute":       "2019-03-01",
	"dns":           "2018-05-01",
	"documentdb":    "2019-08-01",
	"keyvault":      "2016-10-01",
	"msi":           "2018-11-30",
	"network":       "2019-07-01",
}

var (
	tenantIDHack   = "13805ec3-a223-47ad-ad65-8b2baf92c0fb"
	tenantUUIDHack = uuid.Must(uuid.FromString(tenantIDHack))
)

type generator struct {
	production bool
}

func newGenerator(production bool) *generator {
	return &generator{
		production: production,
	}
}

func (g *generator) vnet() *arm.Resource {
	vnet := &network.VirtualNetwork{
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{
				AddressPrefixes: &[]string{
					"10.0.0.0/8",
				},
			},
			Subnets: &[]network.Subnet{
				{
					SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
						AddressPrefix: to.StringPtr("10.1.0.0/16"),
						NetworkSecurityGroup: &network.SecurityGroup{
							ID: to.StringPtr("[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-pe-nsg')]"),
						},
						PrivateEndpointNetworkPolicies: to.StringPtr("Disabled"),
					},
					Name: to.StringPtr("rp-pe-subnet"),
				},
			},
		},
		Name:     to.StringPtr("rp-vnet"),
		Type:     to.StringPtr("Microsoft.Network/virtualNetworks"),
		Location: to.StringPtr("[resourceGroup().location]"),
	}

	if g.production {
		*vnet.Subnets = append(*vnet.Subnets,
			network.Subnet{
				SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
					AddressPrefix: to.StringPtr("10.0.0.0/24"),
					NetworkSecurityGroup: &network.SecurityGroup{
						ID: to.StringPtr("[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-nsg')]"),
					},
					ServiceEndpoints: &[]network.ServiceEndpointPropertiesFormat{
						{
							Service:   to.StringPtr("Microsoft.KeyVault"),
							Locations: &[]string{"*"},
						},
						{
							Service:   to.StringPtr("Microsoft.AzureCosmosDB"),
							Locations: &[]string{"*"},
						},
					},
				},
				Name: to.StringPtr("rp-subnet"),
			},
		)
	}

	return &arm.Resource{
		Resource:   vnet,
		APIVersion: apiVersions["network"],
	}
}

func (g *generator) pip() *arm.Resource {
	return &arm.Resource{
		Resource: &network.PublicIPAddress{
			Sku: &network.PublicIPAddressSku{
				Name: network.PublicIPAddressSkuNameStandard,
			},
			PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
				PublicIPAllocationMethod: network.Static,
			},
			Name:     to.StringPtr("rp-pip"),
			Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: apiVersions["network"],
	}
}

func (g *generator) lb() *arm.Resource {
	return &arm.Resource{
		Resource: &network.LoadBalancer{
			Sku: &network.LoadBalancerSku{
				Name: network.LoadBalancerSkuNameStandard,
			},
			LoadBalancerPropertiesFormat: &network.LoadBalancerPropertiesFormat{
				FrontendIPConfigurations: &[]network.FrontendIPConfiguration{
					{
						FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
							PublicIPAddress: &network.PublicIPAddress{
								ID: to.StringPtr("[resourceId('Microsoft.Network/publicIPAddresses', 'rp-pip')]"),
							},
						},
						Name: to.StringPtr("rp-frontend"),
					},
				},
				BackendAddressPools: &[]network.BackendAddressPool{
					{
						Name: to.StringPtr("rp-backend"),
					},
				},
				LoadBalancingRules: &[]network.LoadBalancingRule{
					{
						LoadBalancingRulePropertiesFormat: &network.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &network.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'rp-lb', 'rp-frontend')]"),
							},
							BackendAddressPool: &network.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'rp-lb', 'rp-backend')]"),
							},
							Probe: &network.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'rp-lb', 'rp-probe')]"),
							},
							Protocol:         network.TransportProtocolTCP,
							LoadDistribution: network.LoadDistributionDefault,
							FrontendPort:     to.Int32Ptr(443),
							BackendPort:      to.Int32Ptr(443),
						},
						Name: to.StringPtr("rp-lbrule"),
					},
				},
				Probes: &[]network.Probe{
					{
						ProbePropertiesFormat: &network.ProbePropertiesFormat{
							Protocol:       network.ProbeProtocolHTTPS,
							Port:           to.Int32Ptr(443),
							NumberOfProbes: to.Int32Ptr(2),
							RequestPath:    to.StringPtr("/healthz/ready"),
						},
						Name: to.StringPtr("rp-probe"),
					},
				},
			},
			Name:     to.StringPtr("rp-lb"),
			Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: apiVersions["network"],
		DependsOn: []string{
			"[resourceId('Microsoft.Network/publicIPAddresses', 'rp-pip')]",
		},
	}
}

func (g *generator) vmss() *arm.Resource {
	parts := []string{
		fmt.Sprintf("base64ToString('%s')", base64.StdEncoding.EncodeToString([]byte("set -ex\n\n"))),
	}

	for _, variable := range []string{"pullSecret", "rpImage", "rpImageAuth"} {
		parts = append(parts,
			fmt.Sprintf("'%s=$(base64 -d <<<'''", strings.ToUpper(variable)),
			fmt.Sprintf("base64(parameters('%s'))", variable),
			"''')\n'",
		)
	}

	trailer := base64.StdEncoding.EncodeToString([]byte(`systemctl stop arorp.service || true

yum -y update -x WALinuxAgent

rpm --import https://dl.fedoraproject.org/pub/epel/RPM-GPG-KEY-EPEL-7

yum -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm || true

cat >/etc/yum.repos.d/azure-cli.repo <<'EOF'
[azurecore]
name=azurecore
baseurl=https://packages.microsoft.com/yumrepos/azurecore
enabled=yes
gpgcheck=no
EOF

yum -y install azsec-clamav azsec-monitor azure-mdsd azure-security docker

firewall-cmd --add-port=443/tcp --permanent

if [[ -n "$RPIMAGEAUTH" ]]; then
  mkdir -p /root/.docker

  cat >/root/.docker/config.json <<EOF
{
	"auths": {
		"${RPIMAGE%%/*}": {
			"auth": "$RPIMAGEAUTH"
		}
	}
}
EOF

else
  rm -rf /root/.docker
fi

cat >/etc/sysconfig/arorp <<EOF
RP_IMAGE='$RPIMAGE'
PULL_SECRET='$PULLSECRET'
EOF

cat >/etc/systemd/system/arorp.service <<EOF
[Unit]
After=docker.service
Requires=docker.service

[Service]
EnvironmentFile=/etc/sysconfig/arorp
ExecStartPre=-/usr/bin/docker rm -f %n
ExecStartPre=/usr/bin/docker pull \$RP_IMAGE
ExecStart=/usr/bin/docker run --rm --name %n -p 443:8443 -e PULL_SECRET \$RP_IMAGE rp
ExecStop=/usr/bin/docker stop -t 90 %n
Restart=always

[Install]
WantedBy=multi-user.target
EOF

systemctl enable arorp.service

(sleep 30; reboot) &
`))

	parts = append(parts, "'\n'", fmt.Sprintf("base64ToString('%s')", trailer))

	script := fmt.Sprintf("[base64(concat(%s))]", strings.Join(parts, ","))

	return &arm.Resource{
		Resource: &compute.VirtualMachineScaleSet{
			Sku: &compute.Sku{
				Name:     to.StringPtr(string(compute.VirtualMachineSizeTypesStandardD2sV3)),
				Tier:     to.StringPtr("Standard"),
				Capacity: to.Int64Ptr(3),
			},
			VirtualMachineScaleSetProperties: &compute.VirtualMachineScaleSetProperties{
				UpgradePolicy: &compute.UpgradePolicy{
					Mode: compute.Manual,
				},
				VirtualMachineProfile: &compute.VirtualMachineScaleSetVMProfile{
					OsProfile: &compute.VirtualMachineScaleSetOSProfile{
						ComputerNamePrefix: to.StringPtr("rp-"),
						AdminUsername:      to.StringPtr("cloud-user"),
						LinuxConfiguration: &compute.LinuxConfiguration{
							DisablePasswordAuthentication: to.BoolPtr(true),
							SSH: &compute.SSHConfiguration{
								PublicKeys: &[]compute.SSHPublicKey{
									{
										Path:    to.StringPtr("/home/cloud-user/.ssh/authorized_keys"),
										KeyData: to.StringPtr("[parameters('sshPublicKey')]"),
									},
								},
							},
						},
					},
					StorageProfile: &compute.VirtualMachineScaleSetStorageProfile{
						ImageReference: &compute.ImageReference{
							Publisher: to.StringPtr("RedHat"),
							Offer:     to.StringPtr("RHEL"),
							Sku:       to.StringPtr("7-RAW"),
							Version:   to.StringPtr("latest"),
						},
						OsDisk: &compute.VirtualMachineScaleSetOSDisk{
							CreateOption: compute.DiskCreateOptionTypesFromImage,
							ManagedDisk: &compute.VirtualMachineScaleSetManagedDiskParameters{
								StorageAccountType: compute.StorageAccountTypesPremiumLRS,
							},
						},
					},
					NetworkProfile: &compute.VirtualMachineScaleSetNetworkProfile{
						HealthProbe: &compute.APIEntityReference{
							ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'rp-lb', 'rp-probe')]"),
						},
						NetworkInterfaceConfigurations: &[]compute.VirtualMachineScaleSetNetworkConfiguration{
							{
								Name: to.StringPtr("rp-vmss-nic"),
								VirtualMachineScaleSetNetworkConfigurationProperties: &compute.VirtualMachineScaleSetNetworkConfigurationProperties{
									Primary: to.BoolPtr(true),
									IPConfigurations: &[]compute.VirtualMachineScaleSetIPConfiguration{
										{
											Name: to.StringPtr("rp-vmss-ipconfig"),
											VirtualMachineScaleSetIPConfigurationProperties: &compute.VirtualMachineScaleSetIPConfigurationProperties{
												Subnet: &compute.APIEntityReference{
													ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet')]"),
												},
												Primary: to.BoolPtr(true),
												PublicIPAddressConfiguration: &compute.VirtualMachineScaleSetPublicIPAddressConfiguration{
													Name: to.StringPtr("rp-vmss-pip"),
												},
												LoadBalancerBackendAddressPools: &[]compute.SubResource{
													{
														ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'rp-lb', 'rp-backend')]"),
													},
												},
											},
										},
									},
								},
							},
						},
					},
					ExtensionProfile: &compute.VirtualMachineScaleSetExtensionProfile{
						Extensions: &[]compute.VirtualMachineScaleSetExtension{
							{
								Name: to.StringPtr("rp-vmss-cse"),
								VirtualMachineScaleSetExtensionProperties: &compute.VirtualMachineScaleSetExtensionProperties{
									Publisher:               to.StringPtr("Microsoft.Azure.Extensions"),
									Type:                    to.StringPtr("CustomScript"),
									TypeHandlerVersion:      to.StringPtr("2.0"),
									AutoUpgradeMinorVersion: to.BoolPtr(true),
									Settings:                map[string]interface{}{},
									ProtectedSettings: map[string]interface{}{
										"script": script,
									},
								},
							},
						},
					},
				},
				Overprovision: to.BoolPtr(false),
			},
			Identity: &compute.VirtualMachineScaleSetIdentity{
				Type: compute.ResourceIdentityTypeUserAssigned,
				UserAssignedIdentities: map[string]*compute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue{
					"[resourceId('Microsoft.ManagedIdentity/userAssignedIdentities', 'rp-identity')]": {},
				},
			},
			Name:     to.StringPtr("rp-vmss"),
			Type:     to.StringPtr("Microsoft.Compute/virtualMachineScaleSets"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: apiVersions["compute"],
		DependsOn: []string{
			"[resourceId('Microsoft.Network/virtualNetworks', 'rp-vnet')]",
			"[resourceId('Microsoft.Network/loadBalancers', 'rp-lb')]",
		},
	}
}

func (g *generator) zone() *arm.Resource {
	return &arm.Resource{
		Resource: &dns.Zone{
			ZoneProperties: &dns.ZoneProperties{},
			Name:           to.StringPtr("[parameters('domainName')]"),
			Type:           to.StringPtr("Microsoft.Network/dnsZones"),
			Location:       to.StringPtr("global"),
		},
		APIVersion: apiVersions["dns"],
	}
}

func (g *generator) accessPolicyEntry() keyvault.AccessPolicyEntry {
	return keyvault.AccessPolicyEntry{
		TenantID: &tenantUUIDHack,
		ObjectID: to.StringPtr("[parameters('rpServicePrincipalId')]"),
		Permissions: &keyvault.Permissions{
			Secrets: &[]keyvault.SecretPermissions{
				keyvault.SecretPermissionsGet,
			},
		},
	}
}

func (g *generator) vault() *arm.Resource {
	vault := &keyvault.Vault{
		Properties: &keyvault.VaultProperties{
			TenantID: &tenantUUIDHack,
			Sku: &keyvault.Sku{
				Name:   keyvault.Standard,
				Family: to.StringPtr("A"),
			},
			AccessPolicies: &[]keyvault.AccessPolicyEntry{},
		},
		Name:     to.StringPtr("[parameters('keyvaultName')]"),
		Type:     to.StringPtr("Microsoft.KeyVault/vaults"),
		Location: to.StringPtr("[resourceGroup().location]"),
	}

	if !g.production {
		vault.Properties.AccessPolicies = &[]keyvault.AccessPolicyEntry{
			g.accessPolicyEntry(),
			{
				TenantID: &tenantUUIDHack,
				ObjectID: to.StringPtr("[parameters('adminObjectId')]"),
				Permissions: &keyvault.Permissions{
					Certificates: &[]keyvault.CertificatePermissions{
						keyvault.Create,
						keyvault.Delete,
						keyvault.Deleteissuers,
						keyvault.Get,
						keyvault.Getissuers,
						keyvault.Import,
						keyvault.List,
						keyvault.Listissuers,
						keyvault.Managecontacts,
						keyvault.Manageissuers,
						keyvault.Purge,
						keyvault.Recover,
						keyvault.Setissuers,
						keyvault.Update,
					},
				},
			},
		}
	}

	return &arm.Resource{
		Resource:   vault,
		APIVersion: apiVersions["keyvault"],
	}
}

func (g *generator) cosmosdb(databaseName string) []*arm.Resource {
	cosmosdb := &documentdb.DatabaseAccountCreateUpdateParameters{
		Kind: documentdb.GlobalDocumentDB,
		DatabaseAccountCreateUpdateProperties: &documentdb.DatabaseAccountCreateUpdateProperties{
			ConsistencyPolicy: &documentdb.ConsistencyPolicy{
				DefaultConsistencyLevel: documentdb.Strong,
			},
			Locations: &[]documentdb.Location{
				{
					LocationName: to.StringPtr("[resourceGroup().location]"),
				},
			},
			DatabaseAccountOfferType:           to.StringPtr(string(documentdb.Standard)),
			DisableKeyBasedMetadataWriteAccess: to.BoolPtr(true),
		},
		Name:     to.StringPtr("[parameters('databaseAccountName')]"),
		Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts"),
		Location: to.StringPtr("[resourceGroup().location]"),
		Tags: map[string]*string{
			"defaultExperience": to.StringPtr("Core (SQL)"),
		},
	}

	r := &arm.Resource{
		Resource:   cosmosdb,
		APIVersion: apiVersions["documentdb"],
	}

	if g.production {
		cosmosdb.IPRangeFilter = to.StringPtr("[concat('104.42.195.92,40.76.54.131,52.176.6.30,52.169.50.45,52.187.184.26', if(equals(parameters('extraCosmosDBIPs'), ''), '', ','), parameters('extraCosmosDBIPs'))]")
		cosmosdb.IsVirtualNetworkFilterEnabled = to.BoolPtr(true)
		cosmosdb.VirtualNetworkRules = &[]documentdb.VirtualNetworkRule{
			{
				ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet')]"),
			},
		}

		r.DependsOn = append(r.DependsOn, "[resourceId('Microsoft.Network/virtualNetworks', 'rp-vnet')]")
	}

	rs := []*arm.Resource{
		r,
	}

	if g.production {
		rs = append(rs, g.database(databaseName, true)...)
	}

	return rs
}

func (g *generator) database(databaseName string, addDependsOn bool) []*arm.Resource {
	var dependsOn []string

	if addDependsOn {
		dependsOn = []string{
			"[resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName'))]",
		}
	}

	return []*arm.Resource{
		{
			Resource: &documentdb.SQLDatabaseCreateUpdateParameters{
				SQLDatabaseCreateUpdateProperties: &documentdb.SQLDatabaseCreateUpdateProperties{
					Resource: &documentdb.SQLDatabaseResource{
						ID: to.StringPtr("[" + databaseName + "]"),
					},
					Options: map[string]*string{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ")]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: apiVersions["documentdb"],
			DependsOn:  dependsOn,
		},
		{
			Resource: &documentdb.SQLContainerCreateUpdateParameters{
				SQLContainerCreateUpdateProperties: &documentdb.SQLContainerCreateUpdateProperties{
					Resource: &documentdb.SQLContainerResource{
						ID: to.StringPtr("AsyncOperations"),
						PartitionKey: &documentdb.ContainerPartitionKey{
							Paths: &[]string{
								"/id",
							},
							Kind: documentdb.PartitionKindHash,
						},
						DefaultTTL: to.Int32Ptr(7 * 86400), // 7 days
					},
					Options: map[string]*string{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/AsyncOperations')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: apiVersions["documentdb"],
			DependsOn:  dependsOn,
		},
		{
			Resource: &documentdb.SQLContainerCreateUpdateParameters{
				SQLContainerCreateUpdateProperties: &documentdb.SQLContainerCreateUpdateProperties{
					Resource: &documentdb.SQLContainerResource{
						ID: to.StringPtr("OpenShiftClusters"),
						PartitionKey: &documentdb.ContainerPartitionKey{
							Paths: &[]string{
								"/partitionKey",
							},
							Kind: documentdb.PartitionKindHash,
						},
						UniqueKeyPolicy: &documentdb.UniqueKeyPolicy{
							UniqueKeys: &[]documentdb.UniqueKey{
								{
									Paths: &[]string{
										"/key",
									},
								},
							},
						},
					},
					Options: map[string]*string{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/OpenShiftClusters')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: apiVersions["documentdb"],
			DependsOn:  dependsOn,
		},
		{
			Resource: &documentdb.SQLContainerCreateUpdateParameters{
				SQLContainerCreateUpdateProperties: &documentdb.SQLContainerCreateUpdateProperties{
					Resource: &documentdb.SQLContainerResource{
						ID: to.StringPtr("Subscriptions"),
						PartitionKey: &documentdb.ContainerPartitionKey{
							Paths: &[]string{
								"/id",
							},
							Kind: documentdb.PartitionKindHash,
						},
					},
					Options: map[string]*string{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Subscriptions')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: apiVersions["documentdb"],
			DependsOn:  dependsOn,
		},
	}
}

func (g *generator) rbac() []*arm.Resource {
	return []*arm.Resource{
		{
			Resource: &authorization.RoleAssignment{
				Name: to.StringPtr("[guid(resourceGroup().id, 'RP / Reader')]"),
				Type: to.StringPtr("Microsoft.Authorization/roleAssignments"),
				RoleAssignmentPropertiesWithScope: &authorization.RoleAssignmentPropertiesWithScope{
					Scope:            to.StringPtr("[resourceGroup().id]"),
					RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', 'acdd72a7-3385-48ef-bd42-f606fba81ae7')]"),
					PrincipalID:      to.StringPtr("[parameters('rpServicePrincipalId')]"),
					PrincipalType:    authorization.ServicePrincipal,
				},
			},
			APIVersion: apiVersions["authorization"],
		},
		{
			Resource: &authorization.RoleAssignment{
				Name: to.StringPtr("[guid(resourceGroup().id, 'RP / Network Contributor')]"),
				Type: to.StringPtr("Microsoft.Authorization/roleAssignments"),
				RoleAssignmentPropertiesWithScope: &authorization.RoleAssignmentPropertiesWithScope{
					Scope:            to.StringPtr("[resourceGroup().id]"),
					RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '4d97b98b-1d4f-4787-a291-c67834d212e7')]"),
					PrincipalID:      to.StringPtr("[parameters('rpServicePrincipalId')]"),
					PrincipalType:    authorization.ServicePrincipal,
				},
			},
			APIVersion: apiVersions["authorization"],
		},
		{
			Resource: &authorization.RoleAssignment{
				Name: to.StringPtr("[concat(parameters('databaseAccountName'), '/Microsoft.Authorization/', guid(resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName')), 'RP / DocumentDB Account Contributor'))]"),
				Type: to.StringPtr("Microsoft.DocumentDB/databaseAccounts/providers/roleAssignments"),
				RoleAssignmentPropertiesWithScope: &authorization.RoleAssignmentPropertiesWithScope{
					Scope:            to.StringPtr("[resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName'))]"),
					RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '5bd9cd88-fe45-4216-938b-f97437e15450')]"),
					PrincipalID:      to.StringPtr("[parameters('rpServicePrincipalId')]"),
					PrincipalType:    authorization.ServicePrincipal,
				},
			},
			APIVersion: apiVersions["authorization"],
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName'))]",
			},
		},
		{
			Resource: &authorization.RoleAssignment{
				Name: to.StringPtr("[concat(parameters('domainName'), '/Microsoft.Authorization/', guid(resourceId('Microsoft.Network/dnsZones', parameters('domainName')), 'RP / DNS Zone Contributor'))]"),
				Type: to.StringPtr("Microsoft.Network/dnsZones/providers/roleAssignments"),
				RoleAssignmentPropertiesWithScope: &authorization.RoleAssignmentPropertiesWithScope{
					Scope:            to.StringPtr("[resourceId('Microsoft.Network/dnsZones', parameters('domainName'))]"),
					RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', 'befefa01-2a29-4197-83a8-272ff33ce314')]"),
					PrincipalID:      to.StringPtr("[parameters('rpServicePrincipalId')]"),
					PrincipalType:    authorization.ServicePrincipal,
				},
			},
			APIVersion: apiVersions["authorization"],
			DependsOn: []string{
				"[resourceId('Microsoft.Network/dnsZones', parameters('domainName'))]",
			},
		},
	}
}

func (g *generator) template() *arm.Template {
	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Parameters:     map[string]*arm.TemplateParameter{},
	}

	if g.production {
		t.Variables = map[string]interface{}{
			"keyvaultAccessPolicies": []keyvault.AccessPolicyEntry{
				g.accessPolicyEntry(),
			},
		}
	}

	params := []string{
		"databaseAccountName",
		"domainName",
		"keyvaultName",
		"rpServicePrincipalId",
	}
	if g.production {
		params = append(params, "pullSecret", "rpImage", "rpImageAuth", "sshPublicKey")
	} else {
		params = append(params, "adminObjectId")
	}

	for _, param := range params {
		typ := "string"
		switch param {
		case "pullSecret", "rpImageAuth":
			typ = "securestring"
		}
		t.Parameters[param] = &arm.TemplateParameter{Type: typ}
	}

	if g.production {
		t.Parameters["extraCosmosDBIPs"] = &arm.TemplateParameter{
			Type:         "string",
			DefaultValue: "",
		}
		t.Parameters["extraKeyvaultAccessPolicies"] = &arm.TemplateParameter{
			Type:         "array",
			DefaultValue: []keyvault.AccessPolicyEntry{},
		}
	}

	if g.production {
		t.Resources = append(t.Resources, g.pip(), g.lb(), g.vmss())
	}
	t.Resources = append(t.Resources, g.zone(), g.vault(), g.vnet())
	if g.production {
		t.Resources = append(t.Resources, g.cosmosdb("'ARO'")...)
	} else {
		t.Resources = append(t.Resources, g.cosmosdb("parameters('databaseName')")...)
	}
	t.Resources = append(t.Resources, g.rbac()...)

	return t
}

func GenerateRPTemplates() error {
	for _, i := range []struct {
		templateFile string
		g            *generator
	}{
		{
			templateFile: "rp-development.json",
			g:            newGenerator(false),
		},
		{
			templateFile: "rp-production.json",
			g:            newGenerator(true),
		},
	} {
		b, err := json.MarshalIndent(i.g.template(), "", "    ")
		if err != nil {
			return err
		}

		// :-(
		b = bytes.ReplaceAll(b, []byte(tenantIDHack), []byte("[subscription().tenantId]"))
		b = bytes.ReplaceAll(b, []byte(`"accessPolicies": []`), []byte(`"accessPolicies": "[concat(variables('keyvaultAccessPolicies'), parameters('extraKeyvaultAccessPolicies'))]"`))

		b = append(b, byte('\n'))

		err = ioutil.WriteFile(i.templateFile, b, 0666)
		if err != nil {
			return err
		}
	}

	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Parameters: map[string]*arm.TemplateParameter{
			"databaseAccountName": {
				Type: "string",
			},
			"databaseName": {
				Type: "string",
			},
		},
	}

	g := newGenerator(false)

	t.Resources = append(t.Resources, g.database("parameters('databaseName')", false)...)

	b, err := json.MarshalIndent(t, "", "    ")
	if err != nil {
		return err
	}

	b = append(b, byte('\n'))

	return ioutil.WriteFile("databases-development.json", b, 0666)
}

func GenerateRPParameterTemplate() error {
	t := newGenerator(true).template()

	p := &arm.Parameters{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentParameters.json#",
		ContentVersion: "1.0.0.0",
		Parameters:     map[string]*arm.ParametersParameter{},
	}

	for name, tp := range t.Parameters {
		param := &arm.ParametersParameter{Value: tp.DefaultValue}
		if param.Value == nil {
			param.Value = ""
		}
		p.Parameters[name] = param
	}

	b, err := json.MarshalIndent(p, "", "    ")
	if err != nil {
		return err
	}

	b = append(b, byte('\n'))

	err = ioutil.WriteFile("rp-production-parameters.json", b, 0666)
	if err != nil {
		return err
	}

	return nil
}
