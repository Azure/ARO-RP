package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	mgmtdocumentdb "github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2019-08-01/documentdb"
	mgmtdns "github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2016-10-01/keyvault"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
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

const (
	capacityHack = 12345
	tenantIDHack = "13805ec3-a223-47ad-ad65-8b2baf92c0fb"
)

var (
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

// halfPeering configures vnetA to peer with vnetB, two symmetrical configurations have to be applied for a peering to work
func (g *generator) halfPeering(vnetA string, vnetB string) *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.VirtualNetworkPeering{
			VirtualNetworkPeeringPropertiesFormat: &mgmtnetwork.VirtualNetworkPeeringPropertiesFormat{
				AllowVirtualNetworkAccess: to.BoolPtr(true),
				AllowForwardedTraffic:     to.BoolPtr(true),
				AllowGatewayTransit:       to.BoolPtr(false),
				UseRemoteGateways:         to.BoolPtr(false),
				RemoteVirtualNetwork: &mgmtnetwork.SubResource{
					ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.Network/virtualNetworks', '%s')]", vnetB)),
				},
			},
			Name: to.StringPtr(fmt.Sprintf("%s/peering-%s", vnetA, vnetB)),
		},
		APIVersion: apiVersions["network"],
		DependsOn: []string{
			fmt.Sprintf("[resourceId('Microsoft.Network/virtualNetworks', '%s')]", vnetA),
			fmt.Sprintf("[resourceId('Microsoft.Network/virtualNetworks', '%s')]", vnetB),
		},
		Type:     "Microsoft.Network/virtualNetworks/virtualNetworkPeerings",
		Location: "[resourceGroup().location]",
	}
}

func (g *generator) rpvnet() *arm.Resource {
	subnet := mgmtnetwork.Subnet{
		SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
			AddressPrefix: to.StringPtr("10.0.0.0/24"),
			NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
				ID: to.StringPtr("[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-nsg')]"),
			},
		},
		Name: to.StringPtr("rp-subnet"),
	}

	if g.production {
		subnet.ServiceEndpoints = &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
			{
				Service:   to.StringPtr("Microsoft.KeyVault"),
				Locations: &[]string{"*"},
			},
			{
				Service:   to.StringPtr("Microsoft.AzureCosmosDB"),
				Locations: &[]string{"*"},
			},
		}
	}

	return &arm.Resource{
		Resource: &mgmtnetwork.VirtualNetwork{
			VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
				AddressSpace: &mgmtnetwork.AddressSpace{
					AddressPrefixes: &[]string{
						"10.0.0.0/24",
					},
				},
				Subnets: &[]mgmtnetwork.Subnet{
					subnet,
				},
			},
			Name:     to.StringPtr("rp-vnet"),
			Type:     to.StringPtr("Microsoft.Network/virtualNetworks"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: apiVersions["network"],
	}
}

func (g *generator) dummyVM() []*arm.Resource {
	return []*arm.Resource{
		{
			Resource: mgmtnetwork.Interface{
				InterfacePropertiesFormat: &mgmtnetwork.InterfacePropertiesFormat{
					IPConfigurations: &[]mgmtnetwork.InterfaceIPConfiguration{
						{
							InterfaceIPConfigurationPropertiesFormat: &mgmtnetwork.InterfaceIPConfigurationPropertiesFormat{
								Subnet: &mgmtnetwork.Subnet{
									ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-pe-vnet-001', 'rp-pe-subnet')]"),
								},
							},
							Name: to.StringPtr("pipConfig"),
						},
					},
				},
				Name:     to.StringPtr("dummy-nic-001"),
				Type:     to.StringPtr("Microsoft.Network/networkInterfaces"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			DependsOn: []string{
				"[resourceId('Microsoft.Network/virtualNetworks', 'rp-pe-vnet-001')]",
			},
			APIVersion: apiVersions["network"],
		},
		{
			Resource: mgmtcompute.VirtualMachine{
				VirtualMachineProperties: &mgmtcompute.VirtualMachineProperties{
					HardwareProfile: &mgmtcompute.HardwareProfile{
						VMSize: mgmtcompute.VirtualMachineSizeTypesStandardD2sV3,
					},
					StorageProfile: &mgmtcompute.StorageProfile{
						ImageReference: &mgmtcompute.ImageReference{
							Publisher: to.StringPtr("RedHat"),
							Offer:     to.StringPtr("RHEL"),
							Sku:       to.StringPtr("8"),
							Version:   to.StringPtr("latest"),
						},
						OsDisk: &mgmtcompute.OSDisk{
							CreateOption: mgmtcompute.DiskCreateOptionTypesFromImage,
							ManagedDisk: &mgmtcompute.ManagedDiskParameters{
								StorageAccountType: mgmtcompute.StorageAccountTypesPremiumLRS,
							},
						},
					},
					OsProfile: &mgmtcompute.OSProfile{
						ComputerName:  to.StringPtr("dummy-vm-001"),
						AdminUsername: to.StringPtr("cloud-user"),
						LinuxConfiguration: &mgmtcompute.LinuxConfiguration{
							SSH: &mgmtcompute.SSHConfiguration{
								PublicKeys: &[]mgmtcompute.SSHPublicKey{
									{
										Path:    to.StringPtr("/home/cloud-user/.ssh/authorized_keys"),
										KeyData: to.StringPtr("[parameters('sshPublicKey')]"),
									},
								},
							},
						},
					},
					NetworkProfile: &mgmtcompute.NetworkProfile{
						NetworkInterfaces: &[]mgmtcompute.NetworkInterfaceReference{
							{
								NetworkInterfaceReferenceProperties: &mgmtcompute.NetworkInterfaceReferenceProperties{
									Primary: to.BoolPtr(true),
								},
								ID: to.StringPtr("[resourceId('Microsoft.Network/networkInterfaces', 'dummy-nic-001')]"),
							},
						},
					},
				},
				Name:     to.StringPtr("dummy-vm-001"),
				Type:     to.StringPtr("Microsoft.Compute/virtualMachines"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			DependsOn: []string{
				"[resourceId('Microsoft.Network/networkInterfaces', 'dummy-nic-001')]",
			},
			APIVersion: apiVersions["compute"],
		},
	}
}

func (g *generator) pevnet() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.VirtualNetwork{
			VirtualNetworkPropertiesFormat: &mgmtnetwork.VirtualNetworkPropertiesFormat{
				AddressSpace: &mgmtnetwork.AddressSpace{
					AddressPrefixes: &[]string{
						"10.0.4.0/22",
					},
				},
				Subnets: &[]mgmtnetwork.Subnet{
					{
						SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
							AddressPrefix: to.StringPtr("10.0.4.0/22"),
							NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
								ID: to.StringPtr("[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-pe-nsg')]"),
							},
							PrivateEndpointNetworkPolicies: to.StringPtr("Disabled"),
						},
						Name: to.StringPtr("rp-pe-subnet"),
					},
				},
			},
			Name:     to.StringPtr("rp-pe-vnet-001"),
			Type:     to.StringPtr("Microsoft.Network/virtualNetworks"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: apiVersions["network"],
	}
}

func (g *generator) pip() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.PublicIPAddress{
			Sku: &mgmtnetwork.PublicIPAddressSku{
				Name: mgmtnetwork.PublicIPAddressSkuNameStandard,
			},
			PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
				PublicIPAllocationMethod: mgmtnetwork.Static,
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
		Resource: &mgmtnetwork.LoadBalancer{
			Sku: &mgmtnetwork.LoadBalancerSku{
				Name: mgmtnetwork.LoadBalancerSkuNameStandard,
			},
			LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
				FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
					{
						FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
							PublicIPAddress: &mgmtnetwork.PublicIPAddress{
								ID: to.StringPtr("[resourceId('Microsoft.Network/publicIPAddresses', 'rp-pip')]"),
							},
						},
						Name: to.StringPtr("rp-frontend"),
					},
				},
				BackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
					{
						Name: to.StringPtr("rp-backend"),
					},
				},
				LoadBalancingRules: &[]mgmtnetwork.LoadBalancingRule{
					{
						LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'rp-lb', 'rp-frontend')]"),
							},
							BackendAddressPool: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'rp-lb', 'rp-backend')]"),
							},
							Probe: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'rp-lb', 'rp-probe')]"),
							},
							Protocol:         mgmtnetwork.TransportProtocolTCP,
							LoadDistribution: mgmtnetwork.LoadDistributionDefault,
							FrontendPort:     to.Int32Ptr(443),
							BackendPort:      to.Int32Ptr(443),
						},
						Name: to.StringPtr("rp-lbrule"),
					},
				},
				Probes: &[]mgmtnetwork.Probe{
					{
						ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
							Protocol:       mgmtnetwork.ProbeProtocolHTTPS,
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

	for _, variable := range []string{
		"clusterMdmMetricNamespace",
		"clusterMdmMonitoringAccount",
		"pullSecret",
		"rpImage",
		"rpImageAuth",
		"rpMdmFrontendUrl",
		"rpMdmMetricNamespace",
		"rpMdmMonitoringAccount",
		"rpMdsdAccount",
		"rpMdsdConfigVersion",
		"rpMdsdEnvironment",
		"rpMdsdNamespace",
		"rpMode",
	} {
		parts = append(parts,
			fmt.Sprintf("'%s=$(base64 -d <<<'''", strings.ToUpper(variable)),
			fmt.Sprintf("base64(parameters('%s'))", variable),
			"''')\n'",
		)
	}

	parts = append(parts,
		fmt.Sprintf("'LOCATION=$(base64 -d <<<'''"),
		fmt.Sprintf("base64(resourceGroup().location)"),
		"''')\n'",
	)

	parts = append(parts,
		fmt.Sprintf("'RESOURCEGROUPNAME=$(base64 -d <<<'''"),
		fmt.Sprintf("base64(resourceGroup().name)"),
		"''')\n'",
	)

	trailer := base64.StdEncoding.EncodeToString([]byte(`yum -y update -x WALinuxAgent

# avoid "error: db5 error(-30969) from dbenv->open: BDB0091 DB_VERSION_MISMATCH: Database environment version mismatch"
rm -f /var/lib/rpm/__db*

rpm --import https://dl.fedoraproject.org/pub/epel/RPM-GPG-KEY-EPEL-8
rpm --import https://packages.microsoft.com/keys/microsoft.asc
rpm --import https://packages.fluentbit.io/fluentbit.key

yum -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm

cat >/etc/yum.repos.d/azure.repo <<'EOF'
[azure-cli]
name=azure-cli
baseurl=https://packages.microsoft.com/yumrepos/azure-cli
enabled=yes
gpgcheck=yes

[azurecore]
name=azurecore
baseurl=https://packages.microsoft.com/yumrepos/azurecore
enabled=yes
gpgcheck=no
EOF

cat >/etc/yum.repos.d/td-agent-bit.repo <<'EOF'
[td-agent-bit]
name=td-agent-bit
baseurl=https://packages.fluentbit.io/centos/7
enabled=yes
gpgcheck=yes
EOF

yum -y install azsec-clamav azsec-monitor azure-cli azure-mdsd azure-security podman-docker td-agent-bit

firewall-cmd --add-port=443/tcp --permanent

# https://bugzilla.redhat.com/show_bug.cgi?id=1805212
sed -i -e 's/iptables/firewalld/' /etc/cni/net.d/87-podman-bridge.conflist

mkdir /root/.docker
cat >/root/.docker/config.json <<EOF
{
	"auths": {
		"${RPIMAGE%%/*}": {
			"auth": "$RPIMAGEAUTH"
		}
	}
}
EOF

cat >/etc/td-agent-bit/td-agent-bit.conf <<'EOF'
[INPUT]
    Name systemd
    Tag journald

[OUTPUT]
    Name forward
    Port 29230
EOF

az login -i --allow-no-subscriptions

SVCVAULTURI="$(az keyvault list -g "$RESOURCEGROUPNAME" --query "[?tags.vault=='service'].properties.vaultUri" -o tsv)"
az keyvault secret download --file /etc/mdm.pem --id "${SVCVAULTURI}secrets/rp-mdm"
chmod 0600 /etc/mdm.pem

az keyvault secret download --file /etc/mdsd.pem --id "${SVCVAULTURI}secrets/rp-mdsd"
chown syslog:syslog /etc/mdsd.pem
chmod 0600 /etc/mdsd.pem

az logout

cat >/etc/default/mdsd <<EOF
MDSD_ROLE_PREFIX=/var/run/mdsd/default
MDSD_OPTIONS="-A -d -r \$MDSD_ROLE_PREFIX"

export SSL_CERT_FILE=/etc/pki/tls/certs/ca-bundle.crt

export MONITORING_GCS_ENVIRONMENT='$RPMDSDENVIRONMENT'
export MONITORING_GCS_ACCOUNT='$RPMDSDACCOUNT'
export MONITORING_GCS_REGION='$LOCATION'
export MONITORING_GCS_CERT_CERTFILE=/etc/mdsd.pem
export MONITORING_GCS_CERT_KEYFILE=/etc/mdsd.pem
export MONITORING_GCS_NAMESPACE='$RPMDSDNAMESPACE'
export MONITORING_CONFIG_VERSION='$RPMDSDCONFIGVERSION'
export MONITORING_USE_GENEVA_CONFIG_SERVICE=true

export MONITORING_TENANT='$LOCATION'
export MONITORING_ROLE=rp
export MONITORING_ROLE_INSTANCE='$(hostname)'
EOF

cat >/etc/sysconfig/mdm <<EOF
RPMDMFRONTENDURL='$RPMDMFRONTENDURL'
RPMDMIMAGE=arosvc.azurecr.io/genevamdm:master_31
RPMDMSOURCEENVIRONMENT='$LOCATION'
RPMDMSOURCEROLE=rp
RPMDMSOURCEROLEINSTANCE='$(hostname)'
EOF

mkdir /var/etw
cat >/etc/systemd/system/mdm.service <<'EOF'
[Unit]
After=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/mdm
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStartPre=/usr/bin/docker pull $RPMDMIMAGE
ExecStart=/usr/bin/docker run \
  --entrypoint /usr/sbin/MetricsExtension \
  --hostname %H \
  --name %N \
  --rm \
  -v /etc/mdm.pem:/etc/mdm.pem \
  -v /var/etw:/var/etw:z \
  $RPMDMIMAGE \
  -CertFile /etc/mdm.pem \
  -FrontEndUrl $RPMDMFRONTENDURL \
  -Logger Console \
  -PrivateKeyFile /etc/mdm.pem \
  -SourceEnvironment $RPMDMSOURCEENVIRONMENT \
  -SourceRole $RPMDMSOURCEROLE \
  -SourceRoleInstance $RPMDMSOURCEROLEINSTANCE
ExecStop=/usr/bin/docker stop %N
Restart=always

[Install]
WantedBy=multi-user.target
EOF

cat >/etc/sysconfig/aro-rp <<EOF
MDM_ACCOUNT='$RPMDMMONITORINGACCOUNT'
MDM_NAMESPACE='$RPMDMMETRICNAMESPACE'
PULL_SECRET='$PULLSECRET'
RPIMAGE='$RPIMAGE'
RP_MODE='$RPMODE'
EOF

cat >/etc/systemd/system/aro-rp.service <<'EOF'
[Unit]
After=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/aro-rp
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStartPre=/usr/bin/docker pull $RPIMAGE
ExecStart=/usr/bin/docker run \
  --hostname %H \
  --name %N \
  --rm \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -e PULL_SECRET \
  -e RP_MODE \
  -p 443:8443 \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  $RPIMAGE \
  rp
ExecStop=/usr/bin/docker stop -t 3600 %N
TimeoutStopSec="3600"
Restart=always

[Install]
WantedBy=multi-user.target
EOF

cat >/etc/sysconfig/aro-monitor <<EOF
MDM_ACCOUNT='$RPMDMMONITORINGACCOUNT'
MDM_NAMESPACE='$RPMDMMETRICNAMESPACE'
CLUSTER_MDM_ACCOUNT='$CLUSTERMDMMONITORINGACCOUNT'
CLUSTER_MDM_NAMESPACE='$CLUSTERMDMMETRICNAMESPACE'
RPIMAGE='$RPIMAGE'
RP_MODE='$RPMODE'
EOF

cat >/etc/systemd/system/aro-monitor.service <<'EOF'
[Unit]
After=network-online.target

[Service]
EnvironmentFile=/etc/sysconfig/aro-monitor
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStartPre=/usr/bin/docker pull $RPIMAGE
ExecStart=/usr/bin/docker run \
  --hostname %H \
  --name %N \
  --rm \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -e RP_MODE \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  $RPIMAGE \
  monitor
Restart=always

[Install]
WantedBy=multi-user.target
EOF

chcon -R system_u:object_r:var_log_t:s0 /var/opt/microsoft/linuxmonagent

for service in aro-monitor aro-rp auoms azsecd azsecmond mdsd mdm chronyd td-agent-bit; do
  systemctl enable $service.service
done

rm /etc/motd.d/*
>/etc/containers/nodocker

(sleep 30; reboot) &
`))

	parts = append(parts, "'\n'", fmt.Sprintf("base64ToString('%s')", trailer))

	script := fmt.Sprintf("[base64(concat(%s))]", strings.Join(parts, ","))

	return &arm.Resource{
		Resource: &mgmtcompute.VirtualMachineScaleSet{
			Sku: &mgmtcompute.Sku{
				Name:     to.StringPtr(string(mgmtcompute.VirtualMachineSizeTypesStandardD2sV3)),
				Tier:     to.StringPtr("Standard"),
				Capacity: to.Int64Ptr(capacityHack),
			},
			VirtualMachineScaleSetProperties: &mgmtcompute.VirtualMachineScaleSetProperties{
				UpgradePolicy: &mgmtcompute.UpgradePolicy{
					Mode: mgmtcompute.Manual,
				},
				VirtualMachineProfile: &mgmtcompute.VirtualMachineScaleSetVMProfile{
					OsProfile: &mgmtcompute.VirtualMachineScaleSetOSProfile{
						ComputerNamePrefix: to.StringPtr("rp-"),
						AdminUsername:      to.StringPtr("cloud-user"),
						LinuxConfiguration: &mgmtcompute.LinuxConfiguration{
							DisablePasswordAuthentication: to.BoolPtr(true),
							SSH: &mgmtcompute.SSHConfiguration{
								PublicKeys: &[]mgmtcompute.SSHPublicKey{
									{
										Path:    to.StringPtr("/home/cloud-user/.ssh/authorized_keys"),
										KeyData: to.StringPtr("[parameters('sshPublicKey')]"),
									},
								},
							},
						},
					},
					StorageProfile: &mgmtcompute.VirtualMachineScaleSetStorageProfile{
						ImageReference: &mgmtcompute.ImageReference{
							Publisher: to.StringPtr("RedHat"),
							Offer:     to.StringPtr("RHEL"),
							Sku:       to.StringPtr("8"),
							Version:   to.StringPtr("latest"),
						},
						OsDisk: &mgmtcompute.VirtualMachineScaleSetOSDisk{
							CreateOption: mgmtcompute.DiskCreateOptionTypesFromImage,
							ManagedDisk: &mgmtcompute.VirtualMachineScaleSetManagedDiskParameters{
								StorageAccountType: mgmtcompute.StorageAccountTypesPremiumLRS,
							},
						},
					},
					NetworkProfile: &mgmtcompute.VirtualMachineScaleSetNetworkProfile{
						HealthProbe: &mgmtcompute.APIEntityReference{
							ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'rp-lb', 'rp-probe')]"),
						},
						NetworkInterfaceConfigurations: &[]mgmtcompute.VirtualMachineScaleSetNetworkConfiguration{
							{
								Name: to.StringPtr("rp-vmss-nic"),
								VirtualMachineScaleSetNetworkConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
									Primary: to.BoolPtr(true),
									IPConfigurations: &[]mgmtcompute.VirtualMachineScaleSetIPConfiguration{
										{
											Name: to.StringPtr("rp-vmss-ipconfig"),
											VirtualMachineScaleSetIPConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetIPConfigurationProperties{
												Subnet: &mgmtcompute.APIEntityReference{
													ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet')]"),
												},
												Primary: to.BoolPtr(true),
												PublicIPAddressConfiguration: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfiguration{
													Name: to.StringPtr("rp-vmss-pip"),
													VirtualMachineScaleSetPublicIPAddressConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfigurationProperties{
														DNSSettings: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfigurationDNSSettings{
															DomainNameLabel: to.StringPtr("[parameters('vmssDomainNameLabel')]"),
														},
													},
												},
												LoadBalancerBackendAddressPools: &[]mgmtcompute.SubResource{
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
					ExtensionProfile: &mgmtcompute.VirtualMachineScaleSetExtensionProfile{
						Extensions: &[]mgmtcompute.VirtualMachineScaleSetExtension{
							{
								Name: to.StringPtr("rp-vmss-cse"),
								VirtualMachineScaleSetExtensionProperties: &mgmtcompute.VirtualMachineScaleSetExtensionProperties{
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
			Identity: &mgmtcompute.VirtualMachineScaleSetIdentity{
				Type: mgmtcompute.ResourceIdentityTypeUserAssigned,
				UserAssignedIdentities: map[string]*mgmtcompute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue{
					"[resourceId('Microsoft.ManagedIdentity/userAssignedIdentities', 'rp-identity')]": {},
				},
			},
			Name:     to.StringPtr("[concat('rp-vmss-', parameters('vmssName'))]"),
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
		Resource: &mgmtdns.Zone{
			ZoneProperties: &mgmtdns.ZoneProperties{},
			Name:           to.StringPtr("[parameters('domainName')]"),
			Type:           to.StringPtr("Microsoft.Network/dnsZones"),
			Location:       to.StringPtr("global"),
		},
		APIVersion: apiVersions["dns"],
	}
}

func (g *generator) clustersKeyvaultAccessPolicies() []mgmtkeyvault.AccessPolicyEntry {
	return []mgmtkeyvault.AccessPolicyEntry{
		{
			TenantID: &tenantUUIDHack,
			ObjectID: to.StringPtr("[parameters('fpServicePrincipalId')]"),
			Permissions: &mgmtkeyvault.Permissions{
				Secrets: &[]mgmtkeyvault.SecretPermissions{
					mgmtkeyvault.SecretPermissionsGet,
				},
				Certificates: &[]mgmtkeyvault.CertificatePermissions{
					mgmtkeyvault.Create,
					mgmtkeyvault.Delete,
				},
			},
		},
	}
}

func (g *generator) serviceKeyvaultAccessPolicies() []mgmtkeyvault.AccessPolicyEntry {
	return []mgmtkeyvault.AccessPolicyEntry{
		{
			TenantID: &tenantUUIDHack,
			ObjectID: to.StringPtr("[parameters('rpServicePrincipalId')]"),
			Permissions: &mgmtkeyvault.Permissions{
				Secrets: &[]mgmtkeyvault.SecretPermissions{
					mgmtkeyvault.SecretPermissionsGet,
				},
			},
		},
	}
}

func (g *generator) clustersKeyvault() *arm.Resource {
	vault := &mgmtkeyvault.Vault{
		Properties: &mgmtkeyvault.VaultProperties{
			TenantID: &tenantUUIDHack,
			Sku: &mgmtkeyvault.Sku{
				Name:   mgmtkeyvault.Standard,
				Family: to.StringPtr("A"),
			},
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{},
		},
		Name:     to.StringPtr("[concat(parameters('keyvaultPrefix'), '" + kvClusterSuffix + "')]"),
		Type:     to.StringPtr("Microsoft.KeyVault/vaults"),
		Location: to.StringPtr("[resourceGroup().location]"),
		Tags: map[string]*string{
			KeyVaultTagName: to.StringPtr(ClustersKeyVaultTagValue),
		},
	}

	if !g.production {
		*vault.Properties.AccessPolicies = append(g.clustersKeyvaultAccessPolicies(),
			mgmtkeyvault.AccessPolicyEntry{
				TenantID: &tenantUUIDHack,
				ObjectID: to.StringPtr("[parameters('adminObjectId')]"),
				Permissions: &mgmtkeyvault.Permissions{
					Certificates: &[]mgmtkeyvault.CertificatePermissions{
						mgmtkeyvault.Get,
						mgmtkeyvault.List,
					},
				},
			},
		)
	}

	return &arm.Resource{
		Resource:   vault,
		APIVersion: apiVersions["keyvault"],
	}
}

func (g *generator) serviceKeyvault() *arm.Resource {
	vault := &mgmtkeyvault.Vault{
		Properties: &mgmtkeyvault.VaultProperties{
			TenantID: &tenantUUIDHack,
			Sku: &mgmtkeyvault.Sku{
				Name:   mgmtkeyvault.Standard,
				Family: to.StringPtr("A"),
			},
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{},
		},
		Name:     to.StringPtr("[concat(parameters('keyvaultPrefix'), '" + kvServiceSuffix + "')]"),
		Type:     to.StringPtr("Microsoft.KeyVault/vaults"),
		Location: to.StringPtr("[resourceGroup().location]"),
		Tags: map[string]*string{
			KeyVaultTagName: to.StringPtr(ServiceKeyVaultTagValue),
		},
	}

	if !g.production {
		*vault.Properties.AccessPolicies = append(g.serviceKeyvaultAccessPolicies(),
			mgmtkeyvault.AccessPolicyEntry{
				TenantID: &tenantUUIDHack,
				ObjectID: to.StringPtr("[parameters('adminObjectId')]"),
				Permissions: &mgmtkeyvault.Permissions{
					Certificates: &[]mgmtkeyvault.CertificatePermissions{
						mgmtkeyvault.Delete,
						mgmtkeyvault.Get,
						mgmtkeyvault.Import,
						mgmtkeyvault.List,
					},
					Secrets: &[]mgmtkeyvault.SecretPermissions{
						mgmtkeyvault.SecretPermissionsSet,
						mgmtkeyvault.SecretPermissionsList,
					},
				},
			},
		)
	}

	return &arm.Resource{
		Resource:   vault,
		APIVersion: apiVersions["keyvault"],
	}
}

func (g *generator) cosmosdb(databaseName string) []*arm.Resource {
	cosmosdb := &mgmtdocumentdb.DatabaseAccountCreateUpdateParameters{
		Kind: mgmtdocumentdb.GlobalDocumentDB,
		DatabaseAccountCreateUpdateProperties: &mgmtdocumentdb.DatabaseAccountCreateUpdateProperties{
			ConsistencyPolicy: &mgmtdocumentdb.ConsistencyPolicy{
				DefaultConsistencyLevel: mgmtdocumentdb.Strong,
			},
			Locations: &[]mgmtdocumentdb.Location{
				{
					LocationName: to.StringPtr("[resourceGroup().location]"),
				},
			},
			DatabaseAccountOfferType: to.StringPtr(string(mgmtdocumentdb.Standard)),
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
		cosmosdb.VirtualNetworkRules = &[]mgmtdocumentdb.VirtualNetworkRule{
			{
				ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet')]"),
			},
		}
		cosmosdb.DisableKeyBasedMetadataWriteAccess = to.BoolPtr(true)

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

	rs := []*arm.Resource{
		{
			Resource: &mgmtdocumentdb.SQLDatabaseCreateUpdateParameters{
				SQLDatabaseCreateUpdateProperties: &mgmtdocumentdb.SQLDatabaseCreateUpdateProperties{
					Resource: &mgmtdocumentdb.SQLDatabaseResource{
						ID: to.StringPtr("[" + databaseName + "]"),
					},
					Options: map[string]*string{
						"x-ms-offer-throughput": to.StringPtr("400"),
					},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ")]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: apiVersions["documentdb"],
		},
		{
			Resource: &mgmtdocumentdb.SQLContainerCreateUpdateParameters{
				SQLContainerCreateUpdateProperties: &mgmtdocumentdb.SQLContainerCreateUpdateProperties{
					Resource: &mgmtdocumentdb.SQLContainerResource{
						ID: to.StringPtr("AsyncOperations"),
						PartitionKey: &mgmtdocumentdb.ContainerPartitionKey{
							Paths: &[]string{
								"/id",
							},
							Kind: mgmtdocumentdb.PartitionKindHash,
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
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
		},
		{
			Resource: &mgmtdocumentdb.SQLContainerCreateUpdateParameters{
				SQLContainerCreateUpdateProperties: &mgmtdocumentdb.SQLContainerCreateUpdateProperties{
					Resource: &mgmtdocumentdb.SQLContainerResource{
						ID: to.StringPtr("Monitors"),
						PartitionKey: &mgmtdocumentdb.ContainerPartitionKey{
							Paths: &[]string{
								"/id",
							},
							Kind: mgmtdocumentdb.PartitionKindHash,
						},
						DefaultTTL: to.Int32Ptr(-1),
					},
					Options: map[string]*string{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Monitors')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: apiVersions["documentdb"],
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
		},
		{
			Resource: &mgmtdocumentdb.SQLContainerCreateUpdateParameters{
				SQLContainerCreateUpdateProperties: &mgmtdocumentdb.SQLContainerCreateUpdateProperties{
					Resource: &mgmtdocumentdb.SQLContainerResource{
						ID: to.StringPtr("OpenShiftClusters"),
						PartitionKey: &mgmtdocumentdb.ContainerPartitionKey{
							Paths: &[]string{
								"/partitionKey",
							},
							Kind: mgmtdocumentdb.PartitionKindHash,
						},
						UniqueKeyPolicy: &mgmtdocumentdb.UniqueKeyPolicy{
							UniqueKeys: &[]mgmtdocumentdb.UniqueKey{
								{
									Paths: &[]string{
										"/key",
									},
								},
								{
									Paths: &[]string{
										"/clusterResourceGroupIdKey",
									},
								},
								{
									Paths: &[]string{
										"/clientIdKey",
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
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
		},
		{
			Resource: &mgmtdocumentdb.SQLContainerCreateUpdateParameters{
				SQLContainerCreateUpdateProperties: &mgmtdocumentdb.SQLContainerCreateUpdateProperties{
					Resource: &mgmtdocumentdb.SQLContainerResource{
						ID: to.StringPtr("Subscriptions"),
						PartitionKey: &mgmtdocumentdb.ContainerPartitionKey{
							Paths: &[]string{
								"/id",
							},
							Kind: mgmtdocumentdb.PartitionKindHash,
						},
					},
					Options: map[string]*string{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Subscriptions')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: apiVersions["documentdb"],
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
		},
	}

	if addDependsOn {
		for i := range rs {
			rs[i].DependsOn = append(rs[i].DependsOn,
				"[resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName'))]",
			)
		}
	}

	return rs
}

func (g *generator) rbac() []*arm.Resource {
	return []*arm.Resource{
		{
			Resource: &mgmtauthorization.RoleAssignment{
				Name: to.StringPtr("[guid(resourceGroup().id, 'RP / Reader')]"),
				Type: to.StringPtr("Microsoft.Authorization/roleAssignments"),
				RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
					Scope:            to.StringPtr("[resourceGroup().id]"),
					RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', 'acdd72a7-3385-48ef-bd42-f606fba81ae7')]"),
					PrincipalID:      to.StringPtr("[parameters('rpServicePrincipalId')]"),
					PrincipalType:    mgmtauthorization.ServicePrincipal,
				},
			},
			APIVersion: apiVersions["authorization"],
		},
		{
			Resource: &mgmtauthorization.RoleAssignment{
				Name: to.StringPtr("[guid(resourceGroup().id, 'FP / Network Contributor')]"),
				Type: to.StringPtr("Microsoft.Authorization/roleAssignments"),
				RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
					Scope:            to.StringPtr("[resourceGroup().id]"),
					RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '4d97b98b-1d4f-4787-a291-c67834d212e7')]"),
					PrincipalID:      to.StringPtr("[parameters('fpServicePrincipalId')]"),
					PrincipalType:    mgmtauthorization.ServicePrincipal,
				},
			},
			APIVersion: apiVersions["authorization"],
		},
		{
			Resource: &mgmtauthorization.RoleAssignment{
				Name: to.StringPtr("[concat(parameters('databaseAccountName'), '/Microsoft.Authorization/', guid(resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName')), 'RP / DocumentDB Account Contributor'))]"),
				Type: to.StringPtr("Microsoft.DocumentDB/databaseAccounts/providers/roleAssignments"),
				RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
					Scope:            to.StringPtr("[resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName'))]"),
					RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', '5bd9cd88-fe45-4216-938b-f97437e15450')]"),
					PrincipalID:      to.StringPtr("[parameters('rpServicePrincipalId')]"),
					PrincipalType:    mgmtauthorization.ServicePrincipal,
				},
			},
			APIVersion: apiVersions["authorization"],
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName'))]",
			},
		},
		{
			Resource: &mgmtauthorization.RoleAssignment{
				Name: to.StringPtr("[concat(parameters('domainName'), '/Microsoft.Authorization/', guid(resourceId('Microsoft.Network/dnsZones', parameters('domainName')), 'FP / DNS Zone Contributor'))]"),
				Type: to.StringPtr("Microsoft.Network/dnsZones/providers/roleAssignments"),
				RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
					Scope:            to.StringPtr("[resourceId('Microsoft.Network/dnsZones', parameters('domainName'))]"),
					RoleDefinitionID: to.StringPtr("[subscriptionResourceId('Microsoft.Authorization/roleDefinitions', 'befefa01-2a29-4197-83a8-272ff33ce314')]"),
					PrincipalID:      to.StringPtr("[parameters('fpServicePrincipalId')]"),
					PrincipalType:    mgmtauthorization.ServicePrincipal,
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
			"clustersKeyvaultAccessPolicies": g.clustersKeyvaultAccessPolicies(),
			"serviceKeyvaultAccessPolicies":  g.serviceKeyvaultAccessPolicies(),
		}
	}

	params := []string{
		"databaseAccountName",
		"domainName",
		"fpServicePrincipalId",
		"keyvaultPrefix",
		"rpServicePrincipalId",
		"sshPublicKey",
	}
	if g.production {
		params = append(params,
			"clusterMdmMetricNamespace",
			"clusterMdmMonitoringAccount",
			"extraCosmosDBIPs",
			"extraKeyvaultAccessPolicies",
			"pullSecret",
			"rpImage",
			"rpImageAuth",
			"rpMdmFrontendUrl",
			"rpMdmMetricNamespace",
			"rpMdmMonitoringAccount",
			"rpMdsdAccount",
			"rpMdsdConfigVersion",
			"rpMdsdEnvironment",
			"rpMdsdNamespace",
			"rpMode",
			"vmssCount",
			"vmssDomainNameLabel",
			"vmssName",
		)
	} else {
		params = append(params,
			"adminObjectId",
		)
	}

	for _, param := range params {
		p := &arm.TemplateParameter{Type: "string"}
		switch param {
		case "extraCosmosDBIPs", "rpMode":
			p.DefaultValue = ""
		case "extraKeyvaultAccessPolicies":
			p.Type = "array"
			p.DefaultValue = []mgmtkeyvault.AccessPolicyEntry{}
		case "pullSecret", "rpImageAuth":
			p.Type = "securestring"
		case "keyvaultPrefix":
			p.MaxLength = 24 - max(len(kvClusterSuffix), len(kvServiceSuffix))
		case "vmssCount":
			p.Type = "int"
			p.DefaultValue = 3
		}
		t.Parameters[param] = p
	}

	if g.production {
		t.Resources = append(t.Resources, g.pip(), g.lb(), g.vmss())
	}
	// clustersKeyvault must preceed serviceKeyvault due to terrible bytes.Replace below
	t.Resources = append(t.Resources, g.zone(),
		g.clustersKeyvault(), g.serviceKeyvault(),
		g.rpvnet(), g.pevnet(),
		g.halfPeering("rp-vnet", "rp-pe-vnet-001"),
		g.halfPeering("rp-pe-vnet-001", "rp-vnet"))
	t.Resources = append(t.Resources, g.dummyVM()...)
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
			templateFile: fileRPDevelopment,
			g:            newGenerator(false),
		},
		{
			templateFile: FileRPProduction,
			g:            newGenerator(true),
		},
	} {
		b, err := json.MarshalIndent(i.g.template(), "", "    ")
		if err != nil {
			return err
		}

		// :-(
		b = bytes.ReplaceAll(b, []byte(tenantIDHack), []byte("[subscription().tenantId]"))
		if i.g.production {
			b = bytes.Replace(b, []byte(`"accessPolicies": []`), []byte(`"accessPolicies": "[concat(variables('clustersKeyvaultAccessPolicies'), parameters('extraKeyvaultAccessPolicies'))]"`), 1)
			b = bytes.Replace(b, []byte(`"accessPolicies": []`), []byte(`"accessPolicies": "[concat(variables('serviceKeyvaultAccessPolicies'), parameters('extraKeyvaultAccessPolicies'))]"`), 1)
			b = bytes.Replace(b, []byte(`"capacity": `+strconv.Itoa(capacityHack)), []byte(`"capacity": "[parameters('vmssCount')]"`), 1)
		}

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

	return ioutil.WriteFile(fileDatabaseDevelopment, b, 0666)
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

	err = ioutil.WriteFile(fileRPProductionParameters, b, 0666)
	if err != nil {
		return err
	}

	return nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
