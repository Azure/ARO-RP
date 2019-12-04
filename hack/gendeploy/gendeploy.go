package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2019-08-01/documentdb"
	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"

	"github.com/jim-minter/rp/pkg/util/arm"
)

var (
	production = flag.Bool("production", false, "output production template")
	debug      = flag.Bool("debug", false, "debug")
	outputFile = flag.String("o", "", "output file")
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
	production           bool
	debug                bool
	rpServicePrincipalID string
}

func (g *generator) msi() *arm.Resource {
	return &arm.Resource{
		Resource: &msi.Identity{
			Name:     to.StringPtr("rp-identity"),
			Location: to.StringPtr("[parameters('location')]"),
			Type:     "Microsoft.ManagedIdentity/userAssignedIdentities",
		},
		APIVersion: apiVersions["msi"],
	}
}

func (g *generator) nsg() *arm.Resource {
	nsg := &network.SecurityGroup{
		SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
			SecurityRules: &[]network.SecurityRule{
				{
					SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
						Protocol:                 network.SecurityRuleProtocolTCP,
						SourcePortRange:          to.StringPtr("*"),
						DestinationPortRange:     to.StringPtr("443"),
						SourceAddressPrefix:      to.StringPtr("*"),
						DestinationAddressPrefix: to.StringPtr("*"),
						Access:                   network.SecurityRuleAccessAllow,
						Priority:                 to.Int32Ptr(101),
						Direction:                network.SecurityRuleDirectionInbound,
					},
					Name: to.StringPtr("rp_in"),
				},
			},
		},
		Name:     to.StringPtr("rp-nsg"),
		Type:     to.StringPtr("Microsoft.Network/networkSecurityGroups"),
		Location: to.StringPtr("[parameters('location')]"),
	}

	if g.debug {
		*nsg.SecurityGroupPropertiesFormat.SecurityRules = append(*nsg.SecurityGroupPropertiesFormat.SecurityRules, network.SecurityRule{
			SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
				Protocol:                 network.SecurityRuleProtocolTCP,
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("22"),
				SourceAddressPrefix:      to.StringPtr("*"),
				DestinationAddressPrefix: to.StringPtr("*"),
				Access:                   network.SecurityRuleAccessAllow,
				Priority:                 to.Int32Ptr(102),
				Direction:                network.SecurityRuleDirectionInbound,
			},
			Name: to.StringPtr("ssh_in"),
		})
	}

	return &arm.Resource{
		Resource:   nsg,
		APIVersion: apiVersions["network"],
	}
}

func (g *generator) vnet() *arm.Resource {
	return &arm.Resource{
		Resource: &network.VirtualNetwork{
			VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
				AddressSpace: &network.AddressSpace{
					AddressPrefixes: &[]string{
						"10.0.0.0/8",
					},
				},
				Subnets: &[]network.Subnet{
					{
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
				},
			},
			Name:     to.StringPtr("rp-vnet"),
			Type:     to.StringPtr("Microsoft.Network/virtualNetworks"),
			Location: to.StringPtr("[parameters('location')]"),
		},
		APIVersion: apiVersions["network"],
		DependsOn: []string{
			"[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-nsg')]",
		},
	}
}

func (g *generator) pip() *arm.Resource {
	pip := &network.PublicIPAddress{
		Sku: &network.PublicIPAddressSku{
			Name: network.PublicIPAddressSkuNameStandard,
		},
		PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
			PublicIPAllocationMethod: network.Static,
		},
		Name:     to.StringPtr("rp-pip"),
		Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
		Location: to.StringPtr("[parameters('location')]"),
	}

	if g.debug {
		pip.PublicIPAddressPropertiesFormat.DNSSettings = &network.PublicIPAddressDNSSettings{
			DomainNameLabel: to.StringPtr("[parameters('domainNameLabel')]"),
		}
	}

	return &arm.Resource{
		Resource:   pip,
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
							BackendPort:      to.Int32Ptr(8443),
						},
						Name: to.StringPtr("rp-lbrule"),
					},
				},
				Probes: &[]network.Probe{
					{
						ProbePropertiesFormat: &network.ProbePropertiesFormat{
							Protocol:       network.ProbeProtocolTCP,
							Port:           to.Int32Ptr(8443),
							NumberOfProbes: to.Int32Ptr(2),
						},
						Name: to.StringPtr("rp-probe"),
					},
				},
			},
			Name:     to.StringPtr("rp-lb"),
			Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
			Location: to.StringPtr("[parameters('location')]"),
		},
		APIVersion: apiVersions["network"],
		DependsOn: []string{
			"[resourceId('Microsoft.Network/publicIPAddresses', 'rp-pip')]",
		},
	}
}

func (g *generator) vmss() *arm.Resource {
	script := base64.StdEncoding.EncodeToString([]byte(`#!/bin/bash
yum -y update -x WALinuxAgent

rpm --import https://dl.fedoraproject.org/pub/epel/RPM-GPG-KEY-EPEL-7

yum -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm

cat >/etc/yum.repos.d/azure-cli.repo <<'EOF'
[azurecore]
name=azurecore
baseurl=https://packages.microsoft.com/yumrepos/azurecore
enabled=yes
gpgcheck=no
EOF

yum -y install azure-mdsd azure-security azsec-monitor azsec-clamav docker

firewall-cmd --add-port=8443/tcp --permanent
firewall-cmd --reload
`))

	vmss := &compute.VirtualMachineScaleSet{
		Sku: &compute.Sku{
			Name:     to.StringPtr(string(compute.VirtualMachineSizeTypesStandardD2sV3)),
			Tier:     to.StringPtr("Standard"),
			Capacity: to.Int64Ptr(1),
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
		Name:     to.StringPtr("rp-vmss"),
		Type:     to.StringPtr("Microsoft.Compute/virtualMachineScaleSets"),
		Location: to.StringPtr("[parameters('location')]"),
	}

	if g.debug {
		(*(*vmss.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations)[0].VirtualMachineScaleSetNetworkConfigurationProperties.IPConfigurations)[0].PublicIPAddressConfiguration = &compute.VirtualMachineScaleSetPublicIPAddressConfiguration{
			Name: to.StringPtr("rp-vmss-pip"),
		}
	}

	return &arm.Resource{
		Resource:   vmss,
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

func (g *generator) vault() *arm.Resource {
	vault := &keyvault.Vault{
		Properties: &keyvault.VaultProperties{
			TenantID: &tenantUUIDHack,
			Sku: &keyvault.Sku{
				Name:   keyvault.Standard,
				Family: to.StringPtr("A"),
			},
			AccessPolicies: &[]keyvault.AccessPolicyEntry{
				{
					TenantID: &tenantUUIDHack,
					ObjectID: to.StringPtr(g.rpServicePrincipalID),
					Permissions: &keyvault.Permissions{
						Secrets: &[]keyvault.SecretPermissions{
							keyvault.SecretPermissionsGet,
						},
					},
				},
			},
		},
		Name:     to.StringPtr("[parameters('keyvaultName')]"),
		Type:     to.StringPtr("Microsoft.KeyVault/vaults"),
		Location: to.StringPtr("[parameters('location')]"),
	}

	if !g.production || g.debug {
		*vault.Properties.AccessPolicies = append(*vault.Properties.AccessPolicies, keyvault.AccessPolicyEntry{
			TenantID: &tenantUUIDHack,
			ObjectID: to.StringPtr("[parameters('adminObjectId')]"),
			Permissions: &keyvault.Permissions{
				Certificates: &[]keyvault.CertificatePermissions{
					keyvault.Import,
					keyvault.List,
				},
			},
		})
	}

	return &arm.Resource{
		Resource:   vault,
		APIVersion: apiVersions["keyvault"],
	}
}

func (g *generator) cosmosdb() []*arm.Resource {
	cosmosdb := &documentdb.DatabaseAccountCreateUpdateParameters{
		Kind: documentdb.GlobalDocumentDB,
		DatabaseAccountCreateUpdateProperties: &documentdb.DatabaseAccountCreateUpdateProperties{
			ConsistencyPolicy: &documentdb.ConsistencyPolicy{
				DefaultConsistencyLevel: documentdb.Strong,
			},
			Locations: &[]documentdb.Location{
				{
					LocationName: to.StringPtr("[parameters('location')]"),
				},
			},
			DatabaseAccountOfferType:           to.StringPtr(string(documentdb.Standard)),
			DisableKeyBasedMetadataWriteAccess: to.BoolPtr(true),
		},
		Name:     to.StringPtr("[parameters('databaseAccountName')]"),
		Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts"),
		Location: to.StringPtr("[parameters('location')]"),
		Tags: map[string]*string{
			"defaultExperience": to.StringPtr("Core (SQL)"),
		},
	}

	r := &arm.Resource{
		Resource:   cosmosdb,
		APIVersion: apiVersions["documentdb"],
	}

	if g.production {
		cosmosdb.IsVirtualNetworkFilterEnabled = to.BoolPtr(true)
		cosmosdb.VirtualNetworkRules = &[]documentdb.VirtualNetworkRule{
			{
				ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet')]"),
			},
		}

		if g.debug {
			cosmosdb.IPRangeFilter = to.StringPtr("104.42.195.92,40.76.54.131,52.176.6.30,52.169.50.45,52.187.184.26")
		}

		r.DependsOn = append(r.DependsOn, "[resourceId('Microsoft.Network/virtualNetworks', 'rp-vnet')]")
	}

	return []*arm.Resource{
		r,
		{
			Resource: &documentdb.SQLDatabaseCreateUpdateParameters{
				SQLDatabaseCreateUpdateProperties: &documentdb.SQLDatabaseCreateUpdateProperties{
					Resource: &documentdb.SQLDatabaseResource{
						ID: to.StringPtr("ARO"),
					},
					Options: map[string]*string{},
				},
				Name: to.StringPtr("[concat(parameters('databaseAccountName'), '/ARO')]"),
				Type: to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases"),
			},
			APIVersion: apiVersions["documentdb"],
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName'))]",
			},
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
				Name: to.StringPtr("[concat(parameters('databaseAccountName'), '/ARO/OpenShiftClusters')]"),
				Type: to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
			},
			APIVersion: apiVersions["documentdb"],
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), 'ARO')]",
			},
		},
		{
			Resource: &documentdb.SQLContainerCreateUpdateParameters{
				SQLContainerCreateUpdateProperties: &documentdb.SQLContainerCreateUpdateProperties{
					Resource: &documentdb.SQLContainerResource{
						ID: to.StringPtr("Subscriptions"),
						PartitionKey: &documentdb.ContainerPartitionKey{
							Paths: &[]string{
								"/key",
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
				Name: to.StringPtr("[concat(parameters('databaseAccountName'), '/ARO/Subscriptions')]"),
				Type: to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
			},
			APIVersion: apiVersions["documentdb"],
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), 'ARO')]",
			},
		},
	}
}

func (g *generator) rbac() []*arm.Resource {
	rs := []*arm.Resource{
		{
			Resource: &authorization.RoleAssignment{
				Name: to.StringPtr("[guid(resourceGroup().id, 'RP / Reader')]"),
				Type: to.StringPtr("Microsoft.Authorization/roleAssignments"),
				RoleAssignmentPropertiesWithScope: &authorization.RoleAssignmentPropertiesWithScope{
					Scope:            to.StringPtr("[resourceGroup().id]"),
					RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', 'acdd72a7-3385-48ef-bd42-f606fba81ae7')]"),
					PrincipalID:      to.StringPtr(g.rpServicePrincipalID),
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
					PrincipalID:      to.StringPtr(g.rpServicePrincipalID),
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
					PrincipalID:      to.StringPtr(g.rpServicePrincipalID),
					PrincipalType:    authorization.ServicePrincipal,
				},
			},
			APIVersion: apiVersions["authorization"],
			DependsOn: []string{
				"[resourceId('Microsoft.Network/dnsZones', parameters('domainName'))]",
			},
		},
	}

	if g.production {
		for i := range rs {
			rs[i].DependsOn = append(rs[i].DependsOn, "[resourceId('Microsoft.ManagedIdentity/userAssignedIdentities', 'rp-identity')]")
		}
	}

	return rs
}

func newGenerator(production, debug bool) *generator {
	g := &generator{
		production: production,
		debug:      debug,
	}

	if production {
		g.rpServicePrincipalID = "[reference(resourceId('Microsoft.ManagedIdentity/userAssignedIdentities', 'rp-identity'), '2018-11-30').principalId]"
	} else {
		g.rpServicePrincipalID = "[parameters('rpServicePrincipalId')]"
	}

	return g
}

func (g *generator) template() *arm.Template {
	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Parameters:     map[string]*arm.Parameter{},
	}

	params := []string{
		"databaseAccountName",
		"domainName",
		"keyvaultName",
		"location",
	}
	if g.production {
		params = append(params, "sshPublicKey")
		if g.debug {
			params = append(params, "adminObjectId", "domainNameLabel")
		}
	} else {
		params = append(params, "adminObjectId", "rpServicePrincipalId")
	}

	for _, param := range params {
		t.Parameters[param] = &arm.Parameter{Type: "string"}
	}

	if g.production {
		t.Resources = append(t.Resources, g.msi(), g.nsg(), g.vnet(), g.pip(), g.lb(), g.vmss())
	}
	t.Resources = append(t.Resources, g.zone(), g.vault())
	t.Resources = append(t.Resources, g.cosmosdb()...)
	t.Resources = append(t.Resources, g.rbac()...)

	return t
}

func run() error {
	g := newGenerator(*production, *debug)

	b, err := json.MarshalIndent(g.template(), "", "    ")
	if err != nil {
		return err
	}
	b = bytes.ReplaceAll(b, []byte(tenantIDHack), []byte("[subscription().tenantId]")) // :-(
	b = append(b, byte('\n'))

	if *outputFile != "" {
		err = ioutil.WriteFile(*outputFile, b, 0666)
	} else {
		_, err = fmt.Print(string(b))
	}

	return err
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		panic(err)
	}
}
