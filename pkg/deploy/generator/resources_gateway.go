package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"fmt"
	"strings"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	mgmtmsi "github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (g *generator) gatewayManagedIdentity() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtmsi.Identity{
			Type:     to.StringPtr("Microsoft.ManagedIdentity/userAssignedIdentities"),
			Name:     to.StringPtr("[concat('aro-gateway-', resourceGroup().location)]"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.ManagedIdentity"),
	}
}

func (g *generator) gatewaySecurityGroup() *arm.Resource {
	return g.securityGroup("gateway-nsg", nil, g.conditionStanza("deployNSGs"))
}

func (g *generator) gatewayVnet() *arm.Resource {
	return g.virtualNetwork("gateway-vnet", "10.0.8.0/24", &[]mgmtnetwork.Subnet{
		{
			SubnetPropertiesFormat: &mgmtnetwork.SubnetPropertiesFormat{
				AddressPrefix: to.StringPtr("10.0.8.0/24"),
				NetworkSecurityGroup: &mgmtnetwork.SecurityGroup{
					ID: to.StringPtr("[resourceId('Microsoft.Network/networkSecurityGroups', 'gateway-nsg')]"),
				},
				ServiceEndpoints: &[]mgmtnetwork.ServiceEndpointPropertiesFormat{
					{
						Service:   to.StringPtr("Microsoft.AzureCosmosDB"),
						Locations: &[]string{"*"},
					},
					{
						Service:   to.StringPtr("Microsoft.ContainerRegistry"),
						Locations: &[]string{"*"},
					},
					{
						Service:   to.StringPtr("Microsoft.EventHub"),
						Locations: &[]string{"*"},
					},
					{
						Service:   to.StringPtr("Microsoft.Storage"),
						Locations: &[]string{"*"},
					},
					{
						Service:   to.StringPtr("Microsoft.KeyVault"),
						Locations: &[]string{"*"},
					},
				},
				PrivateLinkServiceNetworkPolicies: to.StringPtr("Disabled"),
			},
			Name: to.StringPtr("gateway-subnet"),
		},
	}, nil, []string{"[resourceId('Microsoft.Network/networkSecurityGroups', 'gateway-nsg')]"})
}

func (g *generator) gatewayLB() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.LoadBalancer{
			Sku: &mgmtnetwork.LoadBalancerSku{
				Name: mgmtnetwork.LoadBalancerSkuNameStandard,
			},
			LoadBalancerPropertiesFormat: &mgmtnetwork.LoadBalancerPropertiesFormat{
				FrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
					{
						FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
							Subnet: &mgmtnetwork.Subnet{
								ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'gateway-vnet', 'gateway-subnet')]"),
							},
						},
						Zones: &[]string{},
						Name:  to.StringPtr("gateway-frontend"),
					},
				},
				BackendAddressPools: &[]mgmtnetwork.BackendAddressPool{
					{
						Name: to.StringPtr("gateway-backend"),
					},
				},
				LoadBalancingRules: &[]mgmtnetwork.LoadBalancingRule{
					{
						LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'gateway-lb-internal', 'gateway-frontend')]"),
							},
							BackendAddressPool: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'gateway-lb-internal', 'gateway-backend')]"),
							},
							Probe: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'gateway-lb-internal', 'gateway-probe')]"),
							},
							Protocol:         mgmtnetwork.TransportProtocolTCP,
							LoadDistribution: mgmtnetwork.LoadDistributionDefault,
							FrontendPort:     to.Int32Ptr(443),
							BackendPort:      to.Int32Ptr(443),
						},
						Name: to.StringPtr("gateway-lbrule-https"),
					},
					{
						LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'gateway-lb-internal', 'gateway-frontend')]"),
							},
							BackendAddressPool: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'gateway-lb-internal', 'gateway-backend')]"),
							},
							Probe: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'gateway-lb-internal', 'gateway-probe')]"),
							},
							Protocol:         mgmtnetwork.TransportProtocolTCP,
							LoadDistribution: mgmtnetwork.LoadDistributionDefault,
							FrontendPort:     to.Int32Ptr(80),
							BackendPort:      to.Int32Ptr(80),
						},
						Name: to.StringPtr("gateway-lbrule-http"),
					},
				},
				Probes: &[]mgmtnetwork.Probe{
					{
						ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
							Protocol:       mgmtnetwork.ProbeProtocolHTTP,
							Port:           to.Int32Ptr(80),
							NumberOfProbes: to.Int32Ptr(2),
							RequestPath:    to.StringPtr("/healthz/ready"),
						},
						Name: to.StringPtr("gateway-probe"),
					},
				},
			},
			Name:     to.StringPtr("gateway-lb-internal"),
			Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

func (g *generator) gatewayPLS() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtnetwork.PrivateLinkService{
			PrivateLinkServiceProperties: &mgmtnetwork.PrivateLinkServiceProperties{
				LoadBalancerFrontendIPConfigurations: &[]mgmtnetwork.FrontendIPConfiguration{
					{
						ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'gateway-lb-internal', 'gateway-frontend')]"),
					},
				},
				IPConfigurations: &[]mgmtnetwork.PrivateLinkServiceIPConfiguration{
					{
						PrivateLinkServiceIPConfigurationProperties: &mgmtnetwork.PrivateLinkServiceIPConfigurationProperties{
							Subnet: &mgmtnetwork.Subnet{
								ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'gateway-vnet', 'gateway-subnet')]"),
							},
						},
						Name: to.StringPtr("gateway-pls-001-nic"),
					},
				},
				EnableProxyProtocol: to.BoolPtr(true),
			},
			Name:     to.StringPtr("gateway-pls-001"),
			Type:     to.StringPtr("Microsoft.Network/privateLinkServices"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
		DependsOn: []string{
			"Microsoft.Network/loadBalancers/gateway-lb-internal",
		},
	}
}

func (g *generator) gatewayVMSS() *arm.Resource {
	// TODO: there is a lot of duplication with rpVMSS()

	parts := []string{
		fmt.Sprintf("base64ToString('%s')", base64.StdEncoding.EncodeToString([]byte("set -ex\n\n"))),
	}

	for _, variable := range []string{
		"acrResourceId",
		"azureCloudName",
		"azureSecPackVSATenantId",
		"databaseAccountName",
		"dbtokenClientId",
		"dbtokenUrl",
		"mdmFrontendUrl",
		"mdsdEnvironment",
		"gatewayMdsdConfigVersion",
		"gatewayDomains",
		"gatewayFeatures",
		"keyvaultDNSSuffix",
		"keyvaultPrefix",
		"rpImage",
		"rpMdmAccount",
		"rpMdsdAccount",
		"rpMdsdNamespace",
	} {
		parts = append(parts,
			fmt.Sprintf("'%s=$(base64 -d <<<'''", strings.ToUpper(variable)),
			fmt.Sprintf("base64(parameters('%s'))", variable),
			"''')\n'",
		)
	}

	parts = append(parts,
		"'MDMIMAGE=''"+version.MdmImage("")+"''\n'",
	)

	parts = append(parts,
		"'LOCATION=$(base64 -d <<<'''",
		"base64(resourceGroup().location)",
		"''')\n'",
	)

	parts = append(parts,
		"'SUBSCRIPTIONID=$(base64 -d <<<'''",
		"base64(subscription().subscriptionId)",
		"''')\n'",
	)

	parts = append(parts,
		"'RESOURCEGROUPNAME=$(base64 -d <<<'''",
		"base64(resourceGroup().name)",
		"''')\n'",
	)

	trailer := base64.StdEncoding.EncodeToString([]byte(`
yum -y update -x WALinuxAgent

lvextend -l +50%FREE /dev/rootvg/rootlv
xfs_growfs /

lvextend -l +100%FREE /dev/rootvg/varlv
xfs_growfs /var

# avoid "error: db5 error(-30969) from dbenv->open: BDB0091 DB_VERSION_MISMATCH: Database environment version mismatch"
rm -f /var/lib/rpm/__db*

rpm --import https://dl.fedoraproject.org/pub/epel/RPM-GPG-KEY-EPEL-7
rpm --import https://packages.microsoft.com/keys/microsoft.asc
rpm --import https://packages.fluentbit.io/fluentbit.key

for attempt in {1..5}; do
  yum -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm && break
  if [[ ${attempt} -lt 5 ]]; then sleep 10; else exit 1; fi
done

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
baseurl=https://packages.fluentbit.io/centos/7/$basearch
enabled=yes
gpgcheck=yes
EOF

for attempt in {1..5}; do
yum --enablerepo=rhui-rhel-7-server-rhui-optional-rpms -y install clamav azsec-clamav azsec-monitor azure-cli azure-mdsd azure-security docker openssl-perl td-agent-bit && break
  if [[ ${attempt} -lt 5 ]]; then sleep 10; else exit 1; fi
done

rpm -e $(rpm -qa | grep ^abrt-)
cat >/etc/sysctl.d/01-disable-core.conf <<'EOF'
kernel.core_pattern = |/bin/true
EOF
sysctl --system

firewall-cmd --add-port=80/tcp --permanent
firewall-cmd --add-port=443/tcp --permanent

cat >/etc/td-agent-bit/td-agent-bit.conf <<'EOF'
[INPUT]
	Name systemd
	Tag journald
	Systemd_Filter _COMM=aro

[FILTER]
	Name modify
	Match journald
	Remove_wildcard _
	Remove TIMESTAMP

[OUTPUT]
	Name forward
	Match *
	Port 29230
EOF

export AZURE_CLOUD_NAME=$AZURECLOUDNAME
az login -i --allow-no-subscriptions

# The managed identity that the VM runs as only has a single roleassignment.
# This role assignment is ACRPull which is not necessarily present in the
# subscription we're deploying into.  If the identity does not have any
# role assignments scoped on the subscription we're deploying into, it will
# not show on az login -i, which is why the below line is commented.
# az account set -s "$SUBSCRIPTIONID"

systemctl start docker.service
az acr login --name "$(sed -e 's|.*/||' <<<"$ACRRESOURCEID")"

MDMIMAGE="${RPIMAGE%%/*}/${MDMIMAGE##*/}"
docker pull "$MDMIMAGE"
docker pull "$RPIMAGE"

az logout

cat >/etc/sysconfig/mdm <<EOF
MDMFRONTENDURL='$MDMFRONTENDURL'
MDMIMAGE='$MDMIMAGE'
MDMSOURCEENVIRONMENT='$LOCATION'
MDMSOURCEROLE=gateway
MDMSOURCEROLEINSTANCE='$(hostname)'
EOF

mkdir /var/etw
cat >/etc/systemd/system/mdm.service <<'EOF'
[Unit]
After=docker.service
Requires=docker.service

[Service]
EnvironmentFile=/etc/sysconfig/mdm
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --entrypoint /usr/sbin/MetricsExtension \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  -m 2g \
  -v /etc/mdm.pem:/etc/mdm.pem \
  -v /var/etw:/var/etw:z \
  $MDMIMAGE \
  -CertFile /etc/mdm.pem \
  -FrontEndUrl $MDMFRONTENDURL \
  -Logger Console \
  -LogLevel Warning \
  -PrivateKeyFile /etc/mdm.pem \
  -SourceEnvironment $MDMSOURCEENVIRONMENT \
  -SourceRole $MDMSOURCEROLE \
  -SourceRoleInstance $MDMSOURCEROLEINSTANCE
ExecStop=/usr/bin/docker stop %N
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target
EOF

cat >/etc/sysconfig/aro-gateway <<EOF
ACR_RESOURCE_ID='$ACRRESOURCEID'
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
AZURE_DBTOKEN_CLIENT_ID='$DBTOKENCLIENTID'
DBTOKEN_URL='$DBTOKENURL'
MDM_ACCOUNT="$RPMDMACCOUNT"
MDM_NAMESPACE=Gateway
GATEWAY_DOMAINS='$GATEWAYDOMAINS'
GATEWAY_FEATURES='$GATEWAYFEATURES'
RPIMAGE='$RPIMAGE'
EOF

cat >/etc/systemd/system/aro-gateway.service <<'EOF'
[Unit]
After=docker.service
Requires=docker.service

[Service]
EnvironmentFile=/etc/sysconfig/aro-gateway
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --hostname %H \
  --name %N \
  --rm \
  --cap-drop net_raw \
  -e ACR_RESOURCE_ID \
  -e DATABASE_ACCOUNT_NAME \
  -e AZURE_DBTOKEN_CLIENT_ID \
  -e DBTOKEN_URL \
  -e GATEWAY_DOMAINS \
  -e GATEWAY_FEATURES \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -m 2g \
  -p 80:8080 \
  -p 443:8443 \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  $RPIMAGE \
  gateway
ExecStop=/usr/bin/docker stop -t 3600 %N
TimeoutStopSec=3600
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target
EOF

chcon -R system_u:object_r:var_log_t:s0 /var/opt/microsoft/linuxmonagent

mkdir -p /var/lib/waagent/Microsoft.Azure.KeyVault.Store

for var in "mdsd" "mdm"; do
cat >/etc/systemd/system/download-$var-credentials.service <<EOF
[Unit]
Description=Periodic $var credentials refresh

[Service]
Type=oneshot
ExecStart=/usr/local/bin/download-credentials.sh $var
EOF

cat >/etc/systemd/system/download-$var-credentials.timer <<EOF
[Unit]
Description=Periodic $var credentials refresh

[Timer]
OnBootSec=0min
OnCalendar=0/12:00:00
AccuracySec=5s

[Install]
WantedBy=timers.target
EOF
done

cat >/usr/local/bin/download-credentials.sh <<EOF
#!/bin/bash
set -eu

COMPONENT="\$1"
echo "Download \$COMPONENT credentials"

TEMP_DIR=\$(mktemp -d)
export AZURE_CONFIG_DIR=\$(mktemp -d)
az login -i --allow-no-subscriptions

trap "cleanup" EXIT

cleanup() {
  az logout
  [[ "\$TEMP_DIR" =~ /tmp/.+ ]] && rm -rf \$TEMP_DIR
  [[ "\$AZURE_CONFIG_DIR" =~ /tmp/.+ ]] && rm -rf \$AZURE_CONFIG_DIR
}

if [ "\$COMPONENT" = "mdm" ]; then
  CURRENT_CERT_FILE="/etc/mdm.pem"
elif [ "\$COMPONENT" = "mdsd" ]; then
  CURRENT_CERT_FILE="/var/lib/waagent/Microsoft.Azure.KeyVault.Store/mdsd.pem"
else
  echo Invalid usage && exit 1
fi

SECRET_NAME="gwy-\${COMPONENT}"
NEW_CERT_FILE="\$TEMP_DIR/\$COMPONENT.pem"
for attempt in {1..5}; do
  az keyvault secret download --file \$NEW_CERT_FILE --id "https://$KEYVAULTPREFIX-gwy.$KEYVAULTDNSSUFFIX/secrets/\$SECRET_NAME" && break
  if [[ \$attempt -lt 5 ]]; then sleep 10; else exit 1; fi
done

if [ -f \$NEW_CERT_FILE ]; then
  if [ "\$COMPONENT" = "mdsd" ]; then
    chown syslog:syslog \$NEW_CERT_FILE
  else
    sed -i -ne '1,/END CERTIFICATE/ p' \$NEW_CERT_FILE
  fi
  if ! diff $NEW_CERT_FILE $CURRENT_CERT_FILE >/dev/null 2>&1; then
    chmod 0600 \$NEW_CERT_FILE
    mv \$NEW_CERT_FILE \$CURRENT_CERT_FILE
  fi
else
  echo Failed to refresh certificate for \$COMPONENT && exit 1
fi
EOF

chmod u+x /usr/local/bin/download-credentials.sh

systemctl enable download-mdsd-credentials.timer
systemctl enable download-mdm-credentials.timer

/usr/local/bin/download-credentials.sh mdsd
/usr/local/bin/download-credentials.sh mdm
MDSDCERTIFICATESAN=$(openssl x509 -in /var/lib/waagent/Microsoft.Azure.KeyVault.Store/mdsd.pem -noout -subject | sed -e 's/.*CN=//')

cat >/etc/systemd/system/watch-mdm-credentials.service <<EOF
[Unit]
Description=Watch for changes in mdm.pem and restarts the mdm service

[Service]
Type=oneshot
ExecStart=/usr/bin/systemctl restart mdm.service

[Install]
WantedBy=multi-user.target
EOF

cat >/etc/systemd/system/watch-mdm-credentials.path <<EOF
[Path]
PathModified=/etc/mdm.pem

[Install]
WantedBy=multi-user.target
EOF

systemctl enable watch-mdm-credentials.path
systemctl start watch-mdm-credentials.path

mkdir /etc/systemd/system/mdsd.service.d
cat >/etc/systemd/system/mdsd.service.d/override.conf <<'EOF'
[Unit]
After=network-online.target
EOF

cat >/etc/default/mdsd <<EOF
MDSD_ROLE_PREFIX=/var/run/mdsd/default
MDSD_OPTIONS="-A -d -r \$MDSD_ROLE_PREFIX"

export MONITORING_GCS_ENVIRONMENT='$MDSDENVIRONMENT'
export MONITORING_GCS_ACCOUNT='$RPMDSDACCOUNT'
export MONITORING_GCS_REGION='$LOCATION'
export MONITORING_GCS_AUTH_ID_TYPE=AuthKeyVault
export MONITORING_GCS_AUTH_ID='$MDSDCERTIFICATESAN'
export MONITORING_GCS_NAMESPACE='$RPMDSDNAMESPACE'
export MONITORING_CONFIG_VERSION='$GATEWAYMDSDCONFIGVERSION'
export MONITORING_USE_GENEVA_CONFIG_SERVICE=true

export MONITORING_TENANT='$LOCATION'
export MONITORING_ROLE=gateway
export MONITORING_ROLE_INSTANCE='$(hostname)'
EOF

# setting MONITORING_GCS_AUTH_ID_TYPE=AuthKeyVault seems to have caused mdsd not
# to honour SSL_CERT_FILE any more, heaven only knows why.
mkdir -p /usr/lib/ssl/certs
csplit -f /usr/lib/ssl/certs/cert- -b %03d.pem /etc/pki/tls/certs/ca-bundle.crt /^$/1 {*} >/dev/null
c_rehash /usr/lib/ssl/certs

# we leave clientId blank as long as only 1 managed identity assigned to vmss
# if we have more than 1, we will need to populate with clientId used for off-node scanning
cat >/etc/default/vsa-nodescan-agent.config <<EOF
{
    "Nice": 19,
    "Timeout": 10800,
    "ClientId": "",
    "TenantId": "$AZURESECPACKVSATENANTID",
    "ProcessTimeout": 300,
    "CommandDelay": 0
  }
EOF

# we start a cron job to run every hour to ensure the said directory is accessible
# by the correct user as it gets created by root and may cause a race condition
# where root owns the dir instead of syslog
# TODO: https://msazure.visualstudio.com/AzureRedHatOpenShift/_workitems/edit/12591207
cat >/etc/cron.d/mdsd-chown-workaround <<EOF
SHELL=/bin/bash
PATH=/bin
0 * * * * root chown syslog:syslog /var/opt/microsoft/linuxmonagent/eh/EventNotice/arorplogs*
EOF

for service in aro-gateway auoms azsecd azsecmond mdsd mdm chronyd td-agent-bit; do
  systemctl enable $service.service
done

for scan in baseline clamav software; do
  /usr/local/bin/azsecd config -s $scan -d P1D
done

(sleep 30; reboot) &
`))

	parts = append(parts, "'\n'", fmt.Sprintf("base64ToString('%s')", trailer))

	script := fmt.Sprintf("[base64(concat(%s))]", strings.Join(parts, ","))

	return &arm.Resource{
		Resource: &mgmtcompute.VirtualMachineScaleSet{
			Sku: &mgmtcompute.Sku{
				Name:     to.StringPtr("[parameters('gatewayVmSize')]"),
				Tier:     to.StringPtr("Standard"),
				Capacity: to.Int64Ptr(1339),
			},
			VirtualMachineScaleSetProperties: &mgmtcompute.VirtualMachineScaleSetProperties{
				UpgradePolicy: &mgmtcompute.UpgradePolicy{
					Mode: mgmtcompute.UpgradeModeManual,
				},
				VirtualMachineProfile: &mgmtcompute.VirtualMachineScaleSetVMProfile{
					OsProfile: &mgmtcompute.VirtualMachineScaleSetOSProfile{
						ComputerNamePrefix: to.StringPtr("[concat('gateway-', parameters('vmssName'), '-')]"),
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
						HealthProbe: &mgmtcompute.APIEntityReference{
							ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'gateway-lb-internal', 'gateway-probe')]"),
						},
						NetworkInterfaceConfigurations: &[]mgmtcompute.VirtualMachineScaleSetNetworkConfiguration{
							{
								Name: to.StringPtr("gateway-vmss-nic"),
								VirtualMachineScaleSetNetworkConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
									Primary: to.BoolPtr(true),
									// disabling accelerated networking due to egress issues
									// see icm 271210960 (egress) and 274977072 (accelerated networking team)
									EnableAcceleratedNetworking: to.BoolPtr(false),
									IPConfigurations: &[]mgmtcompute.VirtualMachineScaleSetIPConfiguration{
										{
											Name: to.StringPtr("gateway-vmss-ipconfig"),
											VirtualMachineScaleSetIPConfigurationProperties: &mgmtcompute.VirtualMachineScaleSetIPConfigurationProperties{
												Subnet: &mgmtcompute.APIEntityReference{
													ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'gateway-vnet', 'gateway-subnet')]"),
												},
												Primary: to.BoolPtr(true),
												PublicIPAddressConfiguration: &mgmtcompute.VirtualMachineScaleSetPublicIPAddressConfiguration{
													Name: to.StringPtr("gateway-vmss-pip"),
												},
												LoadBalancerBackendAddressPools: &[]mgmtcompute.SubResource{
													{
														ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'gateway-lb-internal', 'gateway-backend')]"),
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
								Name: to.StringPtr("gateway-vmss-cse"),
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
					DiagnosticsProfile: &mgmtcompute.DiagnosticsProfile{
						BootDiagnostics: &mgmtcompute.BootDiagnostics{
							Enabled:    to.BoolPtr(true),
							StorageURI: to.StringPtr("[concat('https://', parameters('gatewayStorageAccountDomain'), '/')]"),
						},
					},
				},
				Overprovision: to.BoolPtr(false),
			},
			Identity: &mgmtcompute.VirtualMachineScaleSetIdentity{
				Type: mgmtcompute.ResourceIdentityTypeUserAssigned,
				UserAssignedIdentities: map[string]*mgmtcompute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue{
					"[resourceId('Microsoft.ManagedIdentity/userAssignedIdentities', concat('aro-gateway-', resourceGroup().location))]": {},
				},
			},
			Name:     to.StringPtr("[concat('gateway-vmss-', parameters('vmssName'))]"),
			Type:     to.StringPtr("Microsoft.Compute/virtualMachineScaleSets"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Compute"),
		DependsOn: []string{
			"[resourceId('Microsoft.Network/loadBalancers', 'gateway-lb-internal')]",
			"[resourceId('Microsoft.Storage/storageAccounts', substring(parameters('gatewayStorageAccountDomain'), 0, indexOf(parameters('gatewayStorageAccountDomain'), '.')))]",
		},
	}
}

func (g *generator) gatewayKeyvaultAccessPolicies() []mgmtkeyvault.AccessPolicyEntry {
	return []mgmtkeyvault.AccessPolicyEntry{
		{
			TenantID: &tenantUUIDHack,
			ObjectID: to.StringPtr("[parameters('gatewayServicePrincipalId')]"),
			Permissions: &mgmtkeyvault.Permissions{
				Secrets: &[]mgmtkeyvault.SecretPermissions{
					mgmtkeyvault.SecretPermissionsGet,
				},
			},
		},
	}
}

func (g *generator) gatewayKeyvault() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtkeyvault.Vault{
			Properties: &mgmtkeyvault.VaultProperties{
				EnableSoftDelete: to.BoolPtr(true),
				TenantID:         &tenantUUIDHack,
				Sku: &mgmtkeyvault.Sku{
					Name:   mgmtkeyvault.Standard,
					Family: to.StringPtr("A"),
				},
				AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{
					{
						ObjectID: to.StringPtr(gatewayAccessPolicyHack),
					},
				},
			},
			Name:     to.StringPtr("[concat(parameters('keyvaultPrefix'), '" + env.GatewayKeyvaultSuffix + "')]"),
			Type:     to.StringPtr("Microsoft.KeyVault/vaults"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
	}
}

func (g *generator) gatewayRBAC() []*arm.Resource {
	return []*arm.Resource{
		rbac.ResourceRoleAssignment(
			rbac.RoleNetworkContributor,
			"parameters('rpServicePrincipalId')",
			"Microsoft.Network/privateLinkServices",
			"'gateway-pls-001'",
		),
	}
}

func (g *generator) gatewayStorageAccount() *arm.Resource {
	return g.storageAccount("[substring(parameters('gatewayStorageAccountDomain'), 0, indexOf(parameters('gatewayStorageAccountDomain'), '.'))]", nil)
}
