package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"fmt"
	"strings"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

func (g *generator) devProxyVMSS() *arm.Resource {
	parts := []string{
		fmt.Sprintf("base64ToString('%s')", base64.StdEncoding.EncodeToString([]byte("set -ex\n\n"))),
	}

	for _, variable := range []string{"proxyImage", "proxyImageAuth"} {
		parts = append(parts,
			fmt.Sprintf("'%s=$(base64 -d <<<'''", strings.ToUpper(variable)),
			fmt.Sprintf("base64(parameters('%s'))", variable),
			"''')\n'",
		)
	}

	for _, variable := range []string{"proxyCert", "proxyClientCert", "proxyKey"} {
		parts = append(parts,
			fmt.Sprintf("'%s='''", strings.ToUpper(variable)),
			fmt.Sprintf("parameters('%s')", variable),
			"'''\n'",
		)
	}

	trailer := base64.StdEncoding.EncodeToString([]byte(`yum -y update -x WALinuxAgent
yum -y install docker

firewall-cmd --add-port=443/tcp --permanent

mkdir /root/.docker
cat >/root/.docker/config.json <<EOF
{
	"auths": {
		"${PROXYIMAGE%%/*}": {
			"auth": "$PROXYIMAGEAUTH"
		}
	}
}
EOF
systemctl start docker.service
docker pull "$PROXYIMAGE"

mkdir /etc/proxy
base64 -d <<<"$PROXYCERT" >/etc/proxy/proxy.crt
base64 -d <<<"$PROXYKEY" >/etc/proxy/proxy.key
base64 -d <<<"$PROXYCLIENTCERT" >/etc/proxy/proxy-client.crt
chown -R 1000:1000 /etc/proxy
chmod 0600 /etc/proxy/proxy.key

cat >/etc/sysconfig/proxy <<EOF
PROXY_IMAGE='$PROXYIMAGE'
EOF

cat >/etc/systemd/system/proxy.service <<'EOF'
[Unit]
After=docker.service
Requires=docker.service

[Service]
EnvironmentFile=/etc/sysconfig/proxy
ExecStartPre=-/usr/bin/docker rm -f %n
ExecStart=/usr/bin/docker run --rm --name %n -p 443:8443 -v /etc/proxy:/secrets $PROXY_IMAGE
ExecStop=/usr/bin/docker stop %n
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target
EOF

systemctl enable proxy.service

(sleep 30; reboot) &
`))

	parts = append(parts, "'\n'", fmt.Sprintf("base64ToString('%s')", trailer))

	script := fmt.Sprintf("[base64(concat(%s))]", strings.Join(parts, ","))

	return &arm.Resource{
		Resource: &mgmtcompute.VirtualMachineScaleSet{
			Sku: &mgmtcompute.Sku{
				Name:     to.StringPtr(string(mgmtcompute.VirtualMachineSizeTypesStandardD2sV3)),
				Tier:     to.StringPtr("Standard"),
				Capacity: to.Int64Ptr(1),
			},
			VirtualMachineScaleSetProperties: &mgmtcompute.VirtualMachineScaleSetProperties{
				UpgradePolicy: &mgmtcompute.UpgradePolicy{
					Mode: mgmtcompute.UpgradeModeManual,
				},
				VirtualMachineProfile: &mgmtcompute.VirtualMachineScaleSetVMProfile{
					OsProfile: &mgmtcompute.VirtualMachineScaleSetOSProfile{
						ComputerNamePrefix: to.StringPtr("dev-proxy-"),
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
							Sku:       to.StringPtr("7-LVM"),
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
						NetworkInterfaceConfigurations: &[]mgmtcompute.VirtualMachineScaleSetNetworkConfiguration{
							{
								Name: to.StringPtr("dev-proxy-vmss-nic"),
								VirtualMachineScaleSetNetworkConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
									Primary: to.BoolPtr(true),
									IPConfigurations: &[]mgmtcompute.VirtualMachineScaleSetIPConfiguration{
										{
											Name: to.StringPtr("dev-proxy-vmss-ipconfig"),
											VirtualMachineScaleSetIPConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetIPConfigurationProperties{
												Subnet: &mgmtcompute.APIEntityReference{
													ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet')]"),
												},
												Primary: to.BoolPtr(true),
												PublicIPAddressConfiguration: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfiguration{
													Name: to.StringPtr("dev-proxy-vmss-pip"),
													VirtualMachineScaleSetPublicIPAddressConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfigurationProperties{
														DNSSettings: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfigurationDNSSettings{
															DomainNameLabel: to.StringPtr("[parameters('proxyDomainNameLabel')]"),
														},
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
								Name: to.StringPtr("dev-proxy-vmss-cse"),
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
			Name:     to.StringPtr("dev-proxy-vmss"),
			Type:     to.StringPtr("Microsoft.Compute/virtualMachineScaleSets"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Compute"),
	}
}

func (g *generator) devVPNPip() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.PublicIPAddress{
			Sku: &mgmtnetwork.PublicIPAddressSku{
				Name: "[parameters('publicIPAddressSkuName')]",
			},
			PublicIPAddressPropertiesFormat: &mgmtnetwork.PublicIPAddressPropertiesFormat{
				PublicIPAllocationMethod: "[parameters('publicIPAddressAllocationMethod')]",
			},
			Name:     to.StringPtr("dev-vpn-pip"),
			Type:     to.StringPtr("Microsoft.Network/publicIPAddresses"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		Condition:  "[equals(parameters('ciCapacity'), 0)]", // TODO(mj): Refactor g.conditionStanza for better usage
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (g *generator) devVnet() *arm.Resource {
	return g.virtualNetwork("dev-vnet", "10.0.0.0/9", &[]mgmtnetwork.Subnet{
		{
			SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
				AddressPrefix: to.StringPtr("10.0.0.0/24"),
			},
			Name: to.StringPtr("GatewaySubnet"),
		},
		{
			SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
				AddressPrefix: to.StringPtr("10.0.1.0/24"),
				NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
					ID: to.StringPtr("[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-nsg')]"),
				},
			},
			Name: to.StringPtr("ToolingSubnet"),
		},
	}, nil, nil)
}

func (g *generator) devVPN() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.VirtualNetworkGateway{
			VirtualNetworkGatewayPropertiesFormat: &mgmtnetwork.VirtualNetworkGatewayPropertiesFormat{
				IPConfigurations: &[]mgmtnetwork.VirtualNetworkGatewayIPConfiguration{
					{
						VirtualNetworkGatewayIPConfigurationPropertiesFormat: &mgmtnetwork.VirtualNetworkGatewayIPConfigurationPropertiesFormat{
							Subnet: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'dev-vnet', 'GatewaySubnet')]"),
							},
							PublicIPAddress: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/publicIPAddresses', 'dev-vpn-pip')]"),
							},
						},
						Name: to.StringPtr("default"),
					},
				},
				VpnType: mgmtnetwork.RouteBased,
				Sku: &mgmtnetwork.VirtualNetworkGatewaySku{
					Name: mgmtnetwork.VirtualNetworkGatewaySkuNameVpnGw1,
					Tier: mgmtnetwork.VirtualNetworkGatewaySkuTierVpnGw1,
				},
				VpnClientConfiguration: &mgmtnetwork.VpnClientConfiguration{
					VpnClientAddressPool: &mgmtnetwork.AddressSpace{
						AddressPrefixes: &[]string{"192.168.255.0/24"},
					},
					VpnClientRootCertificates: &[]mgmtnetwork.VpnClientRootCertificate{
						{
							VpnClientRootCertificatePropertiesFormat: &mgmtnetwork.VpnClientRootCertificatePropertiesFormat{
								PublicCertData: to.StringPtr("[parameters('vpnCACertificate')]"),
							},
							Name: to.StringPtr("dev-vpn-ca"),
						},
					},
					VpnClientProtocols: &[]mgmtnetwork.VpnClientProtocol{
						mgmtnetwork.OpenVPN,
					},
				},
			},
			Name:     to.StringPtr("dev-vpn"),
			Type:     to.StringPtr("Microsoft.Network/virtualNetworkGateways"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		Condition:  "[equals(parameters('ciCapacity'), 0)]", // TODO(mj): Refactor g.conditionStanza for better usage
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
		DependsOn: []string{
			"[resourceId('Microsoft.Network/publicIPAddresses', 'dev-vpn-pip')]",
			"[resourceId('Microsoft.Network/virtualNetworks', 'dev-vnet')]",
		},
	}
}

func (g *generator) devCIPool() *arm.Resource {
	parts := []string{
		fmt.Sprintf("base64ToString('%s')", base64.StdEncoding.EncodeToString([]byte("set -e\n\n"))),
	}

	for _, variable := range []string{
		"ciAzpToken",
		"ciPoolName"} {
		parts = append(parts,
			fmt.Sprintf("'%s='''", strings.ToUpper(variable)),
			fmt.Sprintf("parameters('%s')", variable),
			"'''\n'",
		)
	}

	trailer := base64.StdEncoding.EncodeToString([]byte(`
# Hack - wait on create because the WALinuxAgent sometimes conflicts with the yum update -y below
sleep 60

for attempt in {1..5}; do
  yum -y update -x WALinuxAgent && break
  if [[ ${attempt} -lt 5 ]]; then sleep 10; else exit 1; fi
done

lvextend -l +50%FREE /dev/rootvg/varlv
xfs_growfs /var

lvextend -l +100%FREE /dev/rootvg/homelv
xfs_growfs /home

rpm --import https://dl.fedoraproject.org/pub/epel/RPM-GPG-KEY-EPEL-8
rpm --import https://packages.microsoft.com/keys/microsoft.asc

yum -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm

cat >/etc/yum.repos.d/azure.repo <<'EOF'
[azure-cli]
name=azure-cli
baseurl=https://packages.microsoft.com/yumrepos/azure-cli
enabled=yes
gpgcheck=yes
EOF

yum -y install azure-cli podman podman-docker jq gcc gpgme-devel libassuan-devel git make tmpwatch python3-devel go-toolset-1.14.12-1.module+el8.3.0+8784+380394dc

# Suppress emulation output for podman instead of docker for az acr compatability
mkdir -p /etc/containers/
touch /etc/containers/nodocker

VSTS_AGENT_VERSION=2.193.1
mkdir /home/cloud-user/agent
pushd /home/cloud-user/agent
curl https://vstsagentpackage.azureedge.net/agent/${VSTS_AGENT_VERSION}/vsts-agent-linux-x64-${VSTS_AGENT_VERSION}.tar.gz | tar -xz
chown -R cloud-user:cloud-user .

./bin/installdependencies.sh
sudo -u cloud-user ./config.sh --unattended --url https://dev.azure.com/msazure --auth pat --token "$CIAZPTOKEN" --pool "$CIPOOLNAME" --agent "ARO-RHEL-$HOSTNAME" --replace
./svc.sh install cloud-user
popd

cat >/home/cloud-user/agent/.path <<'EOF'
/usr/local/bin:/usr/bin:/usr/local/sbin:/usr/sbin:/home/cloud-user/.local/bin:/home/cloud-user/bin
EOF

# HACK for XDG_RUNTIME_DIR: https://github.com/containers/podman/issues/427
cat >/home/cloud-user/agent/.env <<'EOF'
GOLANG_FIPS=1
XDG_RUNTIME_DIR=/run/user/1000
EOF

cat >/etc/cron.hourly/tmpwatch <<'EOF'
#!/bin/bash

exec /sbin/tmpwatch 24h /tmp
EOF
chmod +x /etc/cron.hourly/tmpwatch

# HACK - podman doesn't always terminate or clean up it's pause.pid file causing
# 'cannot reexec errors' so attempt to clean it up every minute to keep pipelines running
# smoothly
cat >/usr/local/bin/fix-podman-pause.sh <<'EOF'
#!/bin/bash

PAUSE_FILE='/tmp/podman-run-1000/libpod/tmp/pause.pid'

if [ -f "${PAUSE_FILE}" ]; then
	PID=$(cat ${PAUSE_FILE})
	if ! ps -p $PID > /dev/null; then
		rm $PAUSE_FILE
	fi
fi
EOF
chmod +x /usr/local/bin/fix-podman-pause.sh

# HACK - /tmp will fill up causing build failures
# delete anything not accessed within 2 days
cat >/usr/local/bin/clean-tmp.sh <<'EOF'
#!/bin/bash

find /tmp -type f \( ! -user root \) -atime +2 -delete

EOF
chmod +x /usr/local/bin/clean-tmp.sh

echo "0 0 */1 * * /usr/local/bin/clean-tmp.sh" >> cron
echo "* * * * * /usr/local/bin/fix-podman-pause.sh" >> cron

# HACK - https://github.com/containers/podman/issues/9002
echo "@reboot loginctl enable-linger cloud-user" >> cron

crontab cron
rm cron

(sleep 30; reboot) &
`))

	parts = append(parts, "'\n'", fmt.Sprintf("base64ToString('%s')", trailer))

	script := fmt.Sprintf("[base64(concat(%s))]", strings.Join(parts, ","))

	return &arm.Resource{
		Resource: &mgmtcompute.VirtualMachineScaleSet{
			Sku: &mgmtcompute.Sku{
				Name:     to.StringPtr(string(mgmtcompute.VirtualMachineSizeTypesStandardD2sV3)),
				Tier:     to.StringPtr("Standard"),
				Capacity: to.Int64Ptr(1337),
			},
			VirtualMachineScaleSetProperties: &mgmtcompute.VirtualMachineScaleSetProperties{
				UpgradePolicy: &mgmtcompute.UpgradePolicy{
					Mode: mgmtcompute.UpgradeModeManual,
				},
				VirtualMachineProfile: &mgmtcompute.VirtualMachineScaleSetVMProfile{
					OsProfile: &mgmtcompute.VirtualMachineScaleSetOSProfile{
						ComputerNamePrefix: to.StringPtr("ci-"),
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
							Sku:       to.StringPtr("8-LVM"),
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
						NetworkInterfaceConfigurations: &[]mgmtcompute.VirtualMachineScaleSetNetworkConfiguration{
							{
								Name: to.StringPtr("ci-vmss-nic"),
								VirtualMachineScaleSetNetworkConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
									Primary: to.BoolPtr(true),
									IPConfigurations: &[]mgmtcompute.VirtualMachineScaleSetIPConfiguration{
										{
											Name: to.StringPtr("ci-vmss-ipconfig"),
											VirtualMachineScaleSetIPConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetIPConfigurationProperties{
												Subnet: &mgmtcompute.APIEntityReference{
													ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'dev-vnet', 'ToolingSubnet')]"),
												},
												Primary: to.BoolPtr(true),
												PublicIPAddressConfiguration: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfiguration{
													Name: to.StringPtr("ci-vmss-pip"),
													VirtualMachineScaleSetPublicIPAddressConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfigurationProperties{
														DNSSettings: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfigurationDNSSettings{
															DomainNameLabel: to.StringPtr("aro-ci"),
														},
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
								Name: to.StringPtr("ci-vmss-cse"),
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
			Name:     to.StringPtr("ci-vmss"),
			Type:     to.StringPtr("Microsoft.Compute/virtualMachineScaleSets"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Compute"),
		Condition:  "[greater(parameters('ciCapacity'), 0)]", // TODO(mj): Refactor g.conditionStanza for better usage
		DependsOn: []string{
			"[resourceId('Microsoft.Network/virtualNetworks', 'dev-vnet')]",
		},
	}
}

const (
	sharedKeyVaultName          = "concat(take(resourceGroup().name,15), '" + SharedKeyVaultNameSuffix + "')"
	sharedDiskEncryptionSetName = "concat(resourceGroup().name, '" + SharedDiskEncryptionSetNameSuffix + "')"
	sharedDiskEncryptionKeyName = "concat(resourceGroup().name, '-disk-encryption-key')"

	SharedKeyVaultNameSuffix          = "-sharedKV"
	SharedDiskEncryptionSetNameSuffix = "-disk-encryption-set"
)

// shared keyvault for keys used for disk encryption sets when creating clusters locally
func (g *generator) devDiskEncryptionKeyvault() *arm.Resource {
	return g.keyVault(fmt.Sprintf("[%s]", sharedKeyVaultName), &[]mgmtkeyvault.AccessPolicyEntry{}, nil, nil)
}

func (g *generator) devDiskEncryptionKey() *arm.Resource {
	key := &mgmtkeyvault.Key{
		KeyProperties: &mgmtkeyvault.KeyProperties{
			Kty:     mgmtkeyvault.RSA,
			KeySize: to.Int32Ptr(4096),
		},

		Name:     to.StringPtr(fmt.Sprintf("[concat(%s, '/', %s)]", sharedKeyVaultName, sharedDiskEncryptionKeyName)),
		Type:     to.StringPtr("Microsoft.KeyVault/vaults/keys"),
		Location: to.StringPtr("[resourceGroup().location]"),
	}

	return &arm.Resource{
		Resource:   key,
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
		DependsOn:  []string{fmt.Sprintf("[resourceId('Microsoft.KeyVault/vaults', %s)]", sharedKeyVaultName)},
	}
}

func (g *generator) devDiskEncryptionSet() *arm.Resource {
	diskEncryptionSet := &mgmtcompute.DiskEncryptionSet{
		EncryptionSetProperties: &mgmtcompute.EncryptionSetProperties{
			ActiveKey: &mgmtcompute.KeyVaultAndKeyReference{
				KeyURL: to.StringPtr(fmt.Sprintf("[reference(resourceId('Microsoft.KeyVault/vaults/keys', %s, %s), '%s', 'Full').properties.keyUriWithVersion]", sharedKeyVaultName, sharedDiskEncryptionKeyName, azureclient.APIVersion("Microsoft.KeyVault"))),
				SourceVault: &mgmtcompute.SourceVault{
					ID: to.StringPtr(fmt.Sprintf("[resourceId('Microsoft.KeyVault/vaults', %s)]", sharedKeyVaultName)),
				},
			},
		},

		Name:     to.StringPtr(fmt.Sprintf("[%s]", sharedDiskEncryptionSetName)),
		Type:     to.StringPtr("Microsoft.Compute/diskEncryptionSets"),
		Location: to.StringPtr("[resourceGroup().location]"),
		Identity: &mgmtcompute.EncryptionSetIdentity{Type: mgmtcompute.SystemAssigned},
	}

	return &arm.Resource{
		Resource:   diskEncryptionSet,
		APIVersion: azureclient.APIVersion("Microsoft.Compute"),
		DependsOn: []string{
			fmt.Sprintf("[resourceId('Microsoft.KeyVault/vaults/keys', %s, %s)]", sharedKeyVaultName, sharedDiskEncryptionKeyName),
		},
	}
}

func (g *generator) devDiskEncryptionKeyVaultAccessPolicy() *arm.Resource {
	accessPolicy := &mgmtkeyvault.VaultAccessPolicyParameters{
		Properties: &mgmtkeyvault.VaultAccessPolicyProperties{
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{
				{
					TenantID: &tenantUUIDHack,
					ObjectID: to.StringPtr(fmt.Sprintf("[reference(resourceId('Microsoft.Compute/diskEncryptionSets', %s), '%s', 'Full').identity.PrincipalId]", sharedDiskEncryptionSetName, azureclient.APIVersion("Microsoft.Compute/diskEncryptionSets"))),
					Permissions: &mgmtkeyvault.Permissions{
						Keys: &[]mgmtkeyvault.KeyPermissions{
							mgmtkeyvault.KeyPermissionsGet,
							mgmtkeyvault.KeyPermissionsWrapKey,
							mgmtkeyvault.KeyPermissionsUnwrapKey,
						},
					},
				},
			},
		},

		Name:     to.StringPtr(fmt.Sprintf("[concat(%s, '/add')]", sharedKeyVaultName)),
		Type:     to.StringPtr("Microsoft.KeyVault/vaults/accessPolicies"),
		Location: to.StringPtr("[resourceGroup().location]"),
	}

	return &arm.Resource{
		Resource:   accessPolicy,
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
		DependsOn:  []string{fmt.Sprintf("[resourceId('Microsoft.Compute/diskEncryptionSets', %s)]", sharedDiskEncryptionSetName)},
	}
}
