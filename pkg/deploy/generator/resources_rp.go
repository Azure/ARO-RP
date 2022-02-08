package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/base64"
	"fmt"
	"strings"

	mgmtcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2020-06-01/compute"
	mgmtdocumentdb "github.com/Azure/azure-sdk-for-go/services/cosmos-db/mgmt/2021-01-15/documentdb"
	mgmtkeyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2019-09-01/keyvault"
	mgmtmsi "github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	mgmtcontainerregistry "github.com/Azure/azure-sdk-for-go/services/preview/containerregistry/mgmt/2020-11-01-preview/containerregistry"
	mgmtinsights "github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-03-01/insights"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func (g *generator) rpManagedIdentity() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtmsi.Identity{
			Type:     to.StringPtr("Microsoft.ManagedIdentity/userAssignedIdentities"),
			Name:     to.StringPtr("[concat('aro-rp-', resourceGroup().location)]"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.ManagedIdentity"),
	}
}

func (g *generator) rpSecurityGroup() *arm.Resource {
	rules := []mgmtnetwork.SecurityRule{
		{
			SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
				Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("443"),
				SourceAddressPrefix:      to.StringPtr("AzureResourceManager"),
				DestinationAddressPrefix: to.StringPtr("*"),
				Access:                   mgmtnetwork.SecurityRuleAccessAllow,
				Priority:                 to.Int32Ptr(120),
				Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
			},
			Name: to.StringPtr("rp_in_arm"),
		},
	}

	if !g.production {
		// override production ARM flag for more open configuration in development
		rules[0].SecurityRulePropertiesFormat.SourceAddressPrefix = to.StringPtr("*")

		rules = append(rules, mgmtnetwork.SecurityRule{
			SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
				Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
				SourcePortRange:          to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("22"),
				SourceAddressPrefix:      to.StringPtr("*"),
				DestinationAddressPrefix: to.StringPtr("*"),
				Access:                   mgmtnetwork.SecurityRuleAccessAllow,
				Priority:                 to.Int32Ptr(125),
				Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
			},
			Name: to.StringPtr("ssh_in"),
		})
	} else {
		rules = append(rules,
			mgmtnetwork.SecurityRule{
				SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
					Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
					SourcePortRange:          to.StringPtr("*"),
					DestinationPortRange:     to.StringPtr("445"),
					SourceAddressPrefix:      to.StringPtr("10.0.8.0/24"),
					DestinationAddressPrefix: to.StringPtr("*"),
					Access:                   mgmtnetwork.SecurityRuleAccessAllow,
					Priority:                 to.Int32Ptr(140),
					Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
				},
				Name: to.StringPtr("dbtoken_in_gateway_445"),
			},
			mgmtnetwork.SecurityRule{
				SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
					Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
					SourcePortRange:          to.StringPtr("*"),
					DestinationPortRange:     to.StringPtr("8445"),
					SourceAddressPrefix:      to.StringPtr("10.0.8.0/24"),
					DestinationAddressPrefix: to.StringPtr("*"),
					Access:                   mgmtnetwork.SecurityRuleAccessAllow,
					Priority:                 to.Int32Ptr(141),
					Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
				},
				Name: to.StringPtr("dbtoken_in_gateway_8445"),
			},
			mgmtnetwork.SecurityRule{
				SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
					Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
					SourcePortRange:          to.StringPtr("*"),
					DestinationPortRange:     to.StringPtr("*"),
					SourceAddressPrefix:      to.StringPtr("10.0.8.0/24"),
					DestinationAddressPrefix: to.StringPtr("*"),
					Access:                   mgmtnetwork.SecurityRuleAccessDeny,
					Priority:                 to.Int32Ptr(145),
					Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
				},
				Name: to.StringPtr("deny_in_gateway"),
			},
			mgmtnetwork.SecurityRule{
				SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
					Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
					SourcePortRange:          to.StringPtr("*"),
					DestinationPortRange:     to.StringPtr("443"),
					SourceAddressPrefixes:    to.StringSlicePtr([]string{}),
					Access:                   mgmtnetwork.SecurityRuleAccessAllow,
					DestinationAddressPrefix: to.StringPtr("*"),
					Priority:                 to.Int32Ptr(130),
					Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
				},
				Name: to.StringPtr("rp_in_geneva"),
			},
		)
	}

	return g.securityGroup("rp-nsg", &rules, g.conditionStanza("deployNSGs"))
}

func (g *generator) rpPESecurityGroup() *arm.Resource {
	return g.securityGroup("rp-pe-nsg", nil, g.conditionStanza("deployNSGs"))
}

func (g *generator) rpVnet() *arm.Resource {
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

	return g.virtualNetwork("rp-vnet", "10.0.0.0/24", &[]mgmtnetwork.Subnet{subnet}, nil, []string{"[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-nsg')]"})
}

func (g *generator) rpPEVnet() *arm.Resource {
	return g.virtualNetwork("rp-pe-vnet-001", "10.0.4.0/22", &[]mgmtnetwork.Subnet{
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
	}, nil, []string{"[resourceId('Microsoft.Network/networkSecurityGroups', 'rp-pe-nsg')]"})
}

func (g *generator) rpLB() *arm.Resource {
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
					{
						FrontendIPConfigurationPropertiesFormat: &mgmtnetwork.FrontendIPConfigurationPropertiesFormat{
							PublicIPAddress: &mgmtnetwork.PublicIPAddress{
								ID: to.StringPtr("[resourceId('Microsoft.Network/publicIPAddresses', 'portal-pip')]"),
							},
						},
						Name: to.StringPtr("portal-frontend"),
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
					{
						LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'rp-lb', 'portal-frontend')]"),
							},
							BackendAddressPool: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'rp-lb', 'rp-backend')]"),
							},
							Probe: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'rp-lb', 'portal-probe-https')]"),
							},
							Protocol:         mgmtnetwork.TransportProtocolTCP,
							LoadDistribution: mgmtnetwork.LoadDistributionDefault,
							FrontendPort:     to.Int32Ptr(443),
							BackendPort:      to.Int32Ptr(444),
						},
						Name: to.StringPtr("portal-lbrule"),
					},
					{
						LoadBalancingRulePropertiesFormat: &mgmtnetwork.LoadBalancingRulePropertiesFormat{
							FrontendIPConfiguration: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'rp-lb', 'portal-frontend')]"),
							},
							BackendAddressPool: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'rp-lb', 'rp-backend')]"),
							},
							Probe: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'rp-lb', 'portal-probe-ssh')]"),
							},
							Protocol:         mgmtnetwork.TransportProtocolTCP,
							LoadDistribution: mgmtnetwork.LoadDistributionDefault,
							FrontendPort:     to.Int32Ptr(22),
							BackendPort:      to.Int32Ptr(2222),
						},
						Name: to.StringPtr("portal-lbrule-ssh"),
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
					{
						ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
							Protocol:       mgmtnetwork.ProbeProtocolHTTPS,
							Port:           to.Int32Ptr(444),
							NumberOfProbes: to.Int32Ptr(2),
							RequestPath:    to.StringPtr("/healthz/ready"),
						},
						Name: to.StringPtr("portal-probe-https"),
					},
					{
						ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
							Protocol:       mgmtnetwork.ProbeProtocolTCP,
							Port:           to.Int32Ptr(2222),
							NumberOfProbes: to.Int32Ptr(2),
						},
						Name: to.StringPtr("portal-probe-ssh"),
					},
				},
			},
			Name:     to.StringPtr("rp-lb"),
			Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
		DependsOn: []string{
			"[resourceId('Microsoft.Network/publicIPAddresses', 'portal-pip')]",
			"[resourceId('Microsoft.Network/publicIPAddresses', 'rp-pip')]",
		},
	}
}

func (g *generator) rpLBInternal() *arm.Resource {
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
								ID: to.StringPtr("[resourceId('Microsoft.Network/virtualNetworks/subnets', 'rp-vnet', 'rp-subnet')]"),
							},
						},
						Name:  to.StringPtr("dbtoken-frontend"),
						Zones: &[]string{},
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
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/frontendIPConfigurations', 'rp-lb-internal', 'dbtoken-frontend')]"),
							},
							BackendAddressPool: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'rp-lb-internal', 'rp-backend')]"),
							},
							Probe: &mgmtnetwork.SubResource{
								ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/probes', 'rp-lb-internal', 'dbtoken-probe')]"),
							},
							Protocol:         mgmtnetwork.TransportProtocolTCP,
							LoadDistribution: mgmtnetwork.LoadDistributionDefault,
							FrontendPort:     to.Int32Ptr(8445),
							BackendPort:      to.Int32Ptr(445),
						},
						Name: to.StringPtr("dbtoken-lbrule"),
					},
				},
				Probes: &[]mgmtnetwork.Probe{
					{
						ProbePropertiesFormat: &mgmtnetwork.ProbePropertiesFormat{
							Protocol:       mgmtnetwork.ProbeProtocolHTTPS,
							Port:           to.Int32Ptr(445),
							NumberOfProbes: to.Int32Ptr(2),
							RequestPath:    to.StringPtr("/healthz/ready"),
						},
						Name: to.StringPtr("dbtoken-probe"),
					},
				},
			},
			Name:     to.StringPtr("rp-lb-internal"),
			Type:     to.StringPtr("Microsoft.Network/loadBalancers"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Network"),
	}
}

// rpLBAlert generates an alert resource for the rp-lb healthprobe metric
func (g *generator) rpLBAlert(threshold float64, severity int32, name string, evalFreq string, windowSize string, metric string) *arm.Resource {
	return &arm.Resource{
		Resource: mgmtinsights.MetricAlertResource{
			MetricAlertProperties: &mgmtinsights.MetricAlertProperties{
				Actions: &[]mgmtinsights.MetricAlertAction{
					{
						ActionGroupID: to.StringPtr("[resourceId(parameters('subscriptionResourceGroupName'), 'Microsoft.Insights/actionGroups', 'rp-health-ag')]"),
					},
				},
				Enabled:             to.BoolPtr(true),
				EvaluationFrequency: to.StringPtr(evalFreq),
				Severity:            to.Int32Ptr(severity),
				Scopes: &[]string{
					"[resourceId('Microsoft.Network/loadBalancers', 'rp-lb')]",
				},
				WindowSize:         to.StringPtr(windowSize),
				TargetResourceType: to.StringPtr("Microsoft.Network/loadBalancers"),
				AutoMitigate:       to.BoolPtr(true),
				Criteria: mgmtinsights.MetricAlertSingleResourceMultipleMetricCriteria{
					AllOf: &[]mgmtinsights.MetricCriteria{
						{
							CriterionType:   mgmtinsights.CriterionTypeStaticThresholdCriterion,
							MetricName:      to.StringPtr(metric),
							MetricNamespace: to.StringPtr("microsoft.network/loadBalancers"),
							Name:            to.StringPtr("HealthProbeCheck"),
							Operator:        mgmtinsights.OperatorLessThan,
							Threshold:       to.Float64Ptr(threshold),
							TimeAggregation: mgmtinsights.Average,
						},
					},
					OdataType: mgmtinsights.OdataTypeMicrosoftAzureMonitorSingleResourceMultipleMetricCriteria,
				},
			},
			Name:     to.StringPtr("[concat('" + name + "-', resourceGroup().location)]"),
			Type:     to.StringPtr("Microsoft.Insights/metricAlerts"),
			Location: to.StringPtr("global"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Insights"),
		DependsOn: []string{
			"[resourceId('Microsoft.Network/loadBalancers', 'rp-lb')]",
		},
	}
}

func (g *generator) rpVMSS() *arm.Resource {
	// TODO: there is a lot of duplication with gatewayVMSS() (and other places)

	parts := []string{
		fmt.Sprintf("base64ToString('%s')", base64.StdEncoding.EncodeToString([]byte("set -ex\n\n"))),
	}

	for _, variable := range []string{
		"acrResourceId",
		"adminApiClientCertCommonName",
		"armApiClientCertCommonName",
		"armClientId",
		"azureCloudName",
		"azureSecPackVSATenantId",
		"billingE2EStorageAccountId",
		"clusterMdmAccount",
		"clusterMdsdAccount",
		"clusterMdsdConfigVersion",
		"clusterMdsdNamespace",
		"clusterParentDomainName",
		"databaseAccountName",
		"dbtokenClientId",
		"fluentbitImage",
		"fpClientId",
		"fpServicePrincipalId",
		"gatewayDomains",
		"gatewayResourceGroupName",
		"gatewayServicePrincipalId",
		"keyvaultDNSSuffix",
		"keyvaultPrefix",
		"mdmFrontendUrl",
		"mdsdEnvironment",
		"portalAccessGroupIds",
		"portalClientId",
		"portalElevatedGroupIds",
		"rpFeatures",
		"rpImage",
		"rpMdmAccount",
		"rpMdsdAccount",
		"rpMdsdConfigVersion",
		"rpMdsdNamespace",
		"rpParentDomainName",
	} {
		parts = append(parts,
			fmt.Sprintf("'%s=$(base64 -d <<<'''", strings.ToUpper(variable)),
			fmt.Sprintf("base64(parameters('%s'))", variable),
			"''')\n'",
		)
	}

	for _, variable := range []string{
		"adminApiCaBundle",
		"armApiCaBundle",
	} {
		parts = append(parts,
			fmt.Sprintf("'%s='''", strings.ToUpper(variable)),
			fmt.Sprintf("parameters('%s')", variable),
			"'''\n'",
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

semanage fcontext -a -t var_log_t "/var/log/journal(/.*)?"
mkdir -p /var/log/journal

for attempt in {1..5}; do
yum --enablerepo=rhui-rhel-7-server-rhui-optional-rpms -y install clamav azsec-clamav azsec-monitor azure-cli azure-mdsd azure-security docker openssl-perl && break
  if [[ ${attempt} -lt 5 ]]; then sleep 10; else exit 1; fi
done

rpm -e $(rpm -qa | grep ^abrt-)
cat >/etc/sysctl.d/01-disable-core.conf <<'EOF'
kernel.core_pattern = |/bin/true
EOF
sysctl --system

firewall-cmd --add-port=443/tcp --permanent
firewall-cmd --add-port=444/tcp --permanent
firewall-cmd --add-port=445/tcp --permanent
firewall-cmd --add-port=2222/tcp --permanent

export AZURE_CLOUD_NAME=$AZURECLOUDNAME

az login -i --allow-no-subscriptions

systemctl start docker.service
az acr login --name "$(sed -e 's|.*/||' <<<"$ACRRESOURCEID")"

MDMIMAGE="${RPIMAGE%%/*}/${MDMIMAGE##*/}"
docker pull "$MDMIMAGE"
docker pull "$RPIMAGE"
docker pull "$FLUENTBITIMAGE"

az logout

mkdir -p /etc/fluentbit/
mkdir -p /var/lib/fluent

cat >/etc/fluentbit/fluentbit.conf <<'EOF'
[INPUT]
	Name systemd
	Tag journald
	Systemd_Filter _COMM=aro

[FILTER]
	Name modify
	Match journald
	Remove_wildcard _
	Remove TIMESTAMP

[FILTER]
	Name rewrite_tag
	Match journald
	Rule $LOGKIND ifxaudit ifxaudit false

[OUTPUT]
	Name forward
	Match *
	Port 29230
EOF

echo "FLUENTBITIMAGE=$FLUENTBITIMAGE" >/etc/sysconfig/fluentbit

cat >/etc/systemd/system/fluentbit.service <<'EOF'
[Unit]
After=docker.service
Requires=docker.service
StartLimitIntervalSec=0

[Service]
RestartSec=1s
EnvironmentFile=/etc/sysconfig/fluentbit
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --security-opt label=disable \
  --entrypoint /opt/td-agent-bit/bin/td-agent-bit \
  --net=host \
  --hostname %H \
  --name %N \
  --rm \
  -v /etc/fluentbit/fluentbit.conf:/etc/fluentbit/fluentbit.conf \
  -v /var/lib/fluent:/var/lib/fluent:z \
  -v /var/log/journal:/var/log/journal:ro \
  -v /run/log/journal:/run/log/journal:ro \
  -v /etc/machine-id:/etc/machine-id:ro \
  $FLUENTBITIMAGE \
  -c /etc/fluentbit/fluentbit.conf

ExecStop=/usr/bin/docker stop %N
Restart=always
RestartSec=5
StartLimitInterval=0

[Install]
WantedBy=multi-user.target
EOF

mkdir /etc/aro-rp
base64 -d <<<"$ADMINAPICABUNDLE" >/etc/aro-rp/admin-ca-bundle.pem
if [[ -n "$ARMAPICABUNDLE" ]]; then
  base64 -d <<<"$ARMAPICABUNDLE" >/etc/aro-rp/arm-ca-bundle.pem
fi
chown -R 1000:1000 /etc/aro-rp

cat >/etc/sysconfig/mdm <<EOF
MDMFRONTENDURL='$MDMFRONTENDURL'
MDMIMAGE='$MDMIMAGE'
MDMSOURCEENVIRONMENT='$LOCATION'
MDMSOURCEROLE=rp
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

cat >/etc/sysconfig/aro-rp <<EOF
ACR_RESOURCE_ID='$ACRRESOURCEID'
ADMIN_API_CLIENT_CERT_COMMON_NAME='$ADMINAPICLIENTCERTCOMMONNAME'
ARM_API_CLIENT_CERT_COMMON_NAME='$ARMAPICLIENTCERTCOMMONNAME'
AZURE_ARM_CLIENT_ID='$ARMCLIENTID'
AZURE_FP_CLIENT_ID='$FPCLIENTID'
AZURE_FP_SERVICE_PRINCIPAL_ID='$FPSERVICEPRINCIPALID'
BILLING_E2E_STORAGE_ACCOUNT_ID='$BILLINGE2ESTORAGEACCOUNTID'
CLUSTER_MDSD_ACCOUNT='$CLUSTERMDSDACCOUNT'
CLUSTER_MDSD_CONFIG_VERSION='$CLUSTERMDSDCONFIGVERSION'
CLUSTER_MDSD_NAMESPACE='$CLUSTERMDSDNAMESPACE'
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
DOMAIN_NAME='$LOCATION.$CLUSTERPARENTDOMAINNAME'
GATEWAY_DOMAINS='$GATEWAYDOMAINS'
GATEWAY_RESOURCEGROUP='$GATEWAYRESOURCEGROUPNAME'
KEYVAULT_PREFIX='$KEYVAULTPREFIX'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE=RP
MDSD_ENVIRONMENT='$MDSDENVIRONMENT'
RP_FEATURES='$RPFEATURES'
RPIMAGE='$RPIMAGE'
EOF

cat >/etc/systemd/system/aro-rp.service <<'EOF'
[Unit]
After=docker.service
Requires=docker.service

[Service]
EnvironmentFile=/etc/sysconfig/aro-rp
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --hostname %H \
  --name %N \
  --rm \
  -e ACR_RESOURCE_ID \
  -e ADMIN_API_CLIENT_CERT_COMMON_NAME \
  -e ARM_API_CLIENT_CERT_COMMON_NAME \
  -e AZURE_ARM_CLIENT_ID \
  -e AZURE_FP_CLIENT_ID \
  -e BILLING_E2E_STORAGE_ACCOUNT_ID \
  -e CLUSTER_MDSD_ACCOUNT \
  -e CLUSTER_MDSD_CONFIG_VERSION \
  -e CLUSTER_MDSD_NAMESPACE \
  -e DATABASE_ACCOUNT_NAME \
  -e DOMAIN_NAME \
  -e GATEWAY_DOMAINS \
  -e GATEWAY_RESOURCEGROUP \
  -e KEYVAULT_PREFIX \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -e MDSD_ENVIRONMENT \
  -e RP_FEATURES \
  -m 2g \
  -p 443:8443 \
  -v /etc/aro-rp:/etc/aro-rp \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  $RPIMAGE \
  rp
ExecStop=/usr/bin/docker stop -t 3600 %N
TimeoutStopSec=3600
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target
EOF

cat >/etc/sysconfig/aro-dbtoken <<EOF
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
AZURE_DBTOKEN_CLIENT_ID='$DBTOKENCLIENTID'
AZURE_GATEWAY_SERVICE_PRINCIPAL_ID='$GATEWAYSERVICEPRINCIPALID'
KEYVAULT_PREFIX='$KEYVAULTPREFIX'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE=DBToken
RPIMAGE='$RPIMAGE'
EOF

cat >/etc/systemd/system/aro-dbtoken.service <<'EOF'
[Unit]
After=docker.service
Requires=docker.service

[Service]
EnvironmentFile=/etc/sysconfig/aro-dbtoken
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --hostname %H \
  --name %N \
  --rm \
  -e AZURE_GATEWAY_SERVICE_PRINCIPAL_ID \
  -e DATABASE_ACCOUNT_NAME \
  -e AZURE_DBTOKEN_CLIENT_ID \
  -e KEYVAULT_PREFIX \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -m 2g \
  -p 445:8445 \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  $RPIMAGE \
  dbtoken
ExecStop=/usr/bin/docker stop -t 3600 %N
TimeoutStopSec=3600
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target
EOF

cat >/etc/sysconfig/aro-monitor <<EOF
CLUSTER_MDM_ACCOUNT='$CLUSTERMDMACCOUNT'
CLUSTER_MDM_NAMESPACE=BBM
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
KEYVAULT_PREFIX='$KEYVAULTPREFIX'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE=BBM
RPIMAGE='$RPIMAGE'
EOF

cat >/etc/systemd/system/aro-monitor.service <<'EOF'
[Unit]
After=docker.service
Requires=docker.service

[Service]
EnvironmentFile=/etc/sysconfig/aro-monitor
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --hostname %H \
  --name %N \
  --rm \
  -e CLUSTER_MDM_ACCOUNT \
  -e CLUSTER_MDM_NAMESPACE \
  -e DATABASE_ACCOUNT_NAME \
  -e KEYVAULT_PREFIX \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -m 2g \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  $RPIMAGE \
  monitor
Restart=always
RestartSec=1
StartLimitInterval=0

[Install]
WantedBy=multi-user.target
EOF

cat >/etc/sysconfig/aro-portal <<EOF
AZURE_PORTAL_ACCESS_GROUP_IDS='$PORTALACCESSGROUPIDS'
AZURE_PORTAL_CLIENT_ID='$PORTALCLIENTID'
AZURE_PORTAL_ELEVATED_GROUP_IDS='$PORTALELEVATEDGROUPIDS'
DATABASE_ACCOUNT_NAME='$DATABASEACCOUNTNAME'
KEYVAULT_PREFIX='$KEYVAULTPREFIX'
MDM_ACCOUNT='$RPMDMACCOUNT'
MDM_NAMESPACE=Portal
PORTAL_HOSTNAME='$LOCATION.admin.$RPPARENTDOMAINNAME'
RPIMAGE='$RPIMAGE'
EOF

cat >/etc/systemd/system/aro-portal.service <<'EOF'
[Unit]
After=docker.service
Requires=docker.service
StartLimitInterval=0

[Service]
EnvironmentFile=/etc/sysconfig/aro-portal
ExecStartPre=-/usr/bin/docker rm -f %N
ExecStart=/usr/bin/docker run \
  --hostname %H \
  --name %N \
  --rm \
  -e AZURE_PORTAL_ACCESS_GROUP_IDS \
  -e AZURE_PORTAL_CLIENT_ID \
  -e AZURE_PORTAL_ELEVATED_GROUP_IDS \
  -e DATABASE_ACCOUNT_NAME \
  -e KEYVAULT_PREFIX \
  -e MDM_ACCOUNT \
  -e MDM_NAMESPACE \
  -e PORTAL_HOSTNAME \
  -m 2g \
  -p 444:8444 \
  -p 2222:2222 \
  -v /run/systemd/journal:/run/systemd/journal \
  -v /var/etw:/var/etw:z \
  $RPIMAGE \
  portal
Restart=always
RestartSec=1

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

SECRET_NAME="rp-\${COMPONENT}"
NEW_CERT_FILE="\$TEMP_DIR/\$COMPONENT.pem"
for attempt in {1..5}; do
  az keyvault secret download --file \$NEW_CERT_FILE --id "https://$KEYVAULTPREFIX-svc.$KEYVAULTDNSSUFFIX/secrets/\$SECRET_NAME" && break
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
export MONITORING_CONFIG_VERSION='$RPMDSDCONFIGVERSION'
export MONITORING_USE_GENEVA_CONFIG_SERVICE=true

export MONITORING_TENANT='$LOCATION'
export MONITORING_ROLE=rp
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

for service in aro-dbtoken aro-monitor aro-portal aro-rp auoms azsecd azsecmond mdsd mdm chronyd fluentbit; do
  systemctl enable $service.service
done

for scan in baseline clamav software; do
  /usr/local/bin/azsecd config -s $scan -d P1D
done

restorecon -RF /var/log/*
(sleep 30; reboot) &
`))

	parts = append(parts, "'\n'", fmt.Sprintf("base64ToString('%s')", trailer))

	script := fmt.Sprintf("[base64(concat(%s))]", strings.Join(parts, ","))

	return &arm.Resource{
		Resource: &mgmtcompute.VirtualMachineScaleSet{
			Sku: &mgmtcompute.Sku{
				Name:     to.StringPtr("[parameters('vmSize')]"),
				Tier:     to.StringPtr("Standard"),
				Capacity: to.Int64Ptr(1338),
			},
			VirtualMachineScaleSetProperties: &mgmtcompute.VirtualMachineScaleSetProperties{
				UpgradePolicy: &mgmtcompute.UpgradePolicy{
					Mode: mgmtcompute.UpgradeModeManual,
				},
				VirtualMachineProfile: &mgmtcompute.VirtualMachineScaleSetVMProfile{
					OsProfile: &mgmtcompute.VirtualMachineScaleSetOSProfile{
						ComputerNamePrefix: to.StringPtr("[concat('rp-', parameters('vmssName'), '-')]"),
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
												},
												LoadBalancerBackendAddressPools: &[]mgmtcompute.SubResource{
													{
														ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'rp-lb', 'rp-backend')]"),
													},
													{
														ID: to.StringPtr("[resourceId('Microsoft.Network/loadBalancers/backendAddressPools', 'rp-lb-internal', 'rp-backend')]"),
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
					DiagnosticsProfile: &mgmtcompute.DiagnosticsProfile{
						BootDiagnostics: &mgmtcompute.BootDiagnostics{
							Enabled:    to.BoolPtr(true),
							StorageURI: to.StringPtr("[concat('https://', parameters('storageAccountDomain'), '/')]"),
						},
					},
				},
				Overprovision: to.BoolPtr(false),
			},
			Identity: &mgmtcompute.VirtualMachineScaleSetIdentity{
				Type: mgmtcompute.ResourceIdentityTypeUserAssigned,
				UserAssignedIdentities: map[string]*mgmtcompute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue{
					"[resourceId('Microsoft.ManagedIdentity/userAssignedIdentities', concat('aro-rp-', resourceGroup().location))]": {},
				},
			},
			Name:     to.StringPtr("[concat('rp-vmss-', parameters('vmssName'))]"),
			Type:     to.StringPtr("Microsoft.Compute/virtualMachineScaleSets"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.Compute"),
		DependsOn: []string{
			"[resourceId('Microsoft.Authorization/roleAssignments', guid(resourceGroup().id, parameters('rpServicePrincipalId'), 'RP / Reader'))]",
			"[resourceId('Microsoft.Network/loadBalancers', 'rp-lb')]",
			"[resourceId('Microsoft.Network/loadBalancers', 'rp-lb-internal')]",
			"[resourceId('Microsoft.Storage/storageAccounts', substring(parameters('storageAccountDomain'), 0, indexOf(parameters('storageAccountDomain'), '.')))]",
		},
	}
}

func (g *generator) rpParentDNSZone() *arm.Resource {
	return g.dnsZone("[parameters('rpParentDomainName')]")
}

func (g *generator) rpClusterParentDNSZone() *arm.Resource {
	return g.dnsZone("[parameters('clusterParentDomainName')]")
}

func (g *generator) rpDNSZone() *arm.Resource {
	return g.dnsZone("[concat(resourceGroup().location, '.', parameters('clusterParentDomainName'))]")
}

func (g *generator) rpClusterKeyvaultAccessPolicies() []mgmtkeyvault.AccessPolicyEntry {
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
					mgmtkeyvault.Get,
					mgmtkeyvault.Update,
				},
			},
		},
	}
}

func (g *generator) rpDBTokenKeyvaultAccessPolicies() []mgmtkeyvault.AccessPolicyEntry {
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

func (g *generator) rpPortalKeyvaultAccessPolicies() []mgmtkeyvault.AccessPolicyEntry {
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

func (g *generator) rpServiceKeyvaultAccessPolicies() []mgmtkeyvault.AccessPolicyEntry {
	return []mgmtkeyvault.AccessPolicyEntry{
		{
			TenantID: &tenantUUIDHack,
			ObjectID: to.StringPtr("[parameters('rpServicePrincipalId')]"),
			Permissions: &mgmtkeyvault.Permissions{
				Secrets: &[]mgmtkeyvault.SecretPermissions{
					mgmtkeyvault.SecretPermissionsGet,
					mgmtkeyvault.SecretPermissionsList,
				},
			},
		},
	}
}

func (g *generator) rpClusterKeyvault() *arm.Resource {
	vault := &mgmtkeyvault.Vault{
		Properties: &mgmtkeyvault.VaultProperties{
			EnableSoftDelete: to.BoolPtr(true),
			TenantID:         &tenantUUIDHack,
			Sku: &mgmtkeyvault.Sku{
				Name:   mgmtkeyvault.Standard,
				Family: to.StringPtr("A"),
			},
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{
				{
					ObjectID: to.StringPtr(clusterAccessPolicyHack),
				},
			},
		},
		Name:     to.StringPtr("[concat(parameters('keyvaultPrefix'), '" + env.ClusterKeyvaultSuffix + "')]"),
		Type:     to.StringPtr("Microsoft.KeyVault/vaults"),
		Location: to.StringPtr("[resourceGroup().location]"),
	}

	if !g.production {
		*vault.Properties.AccessPolicies = append(g.rpClusterKeyvaultAccessPolicies(),
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
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
	}
}

func (g *generator) rpDBTokenKeyvault() *arm.Resource {
	vault := &mgmtkeyvault.Vault{
		Properties: &mgmtkeyvault.VaultProperties{
			EnableSoftDelete: to.BoolPtr(true),
			TenantID:         &tenantUUIDHack,
			Sku: &mgmtkeyvault.Sku{
				Name:   mgmtkeyvault.Standard,
				Family: to.StringPtr("A"),
			},
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{
				{
					ObjectID: to.StringPtr(dbTokenAccessPolicyHack),
				},
			},
		},
		Name:     to.StringPtr("[concat(parameters('keyvaultPrefix'), '" + env.DBTokenKeyvaultSuffix + "')]"),
		Type:     to.StringPtr("Microsoft.KeyVault/vaults"),
		Location: to.StringPtr("[resourceGroup().location]"),
	}

	if !g.production {
		*vault.Properties.AccessPolicies = append(g.rpDBTokenKeyvaultAccessPolicies(),
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
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
	}
}

func (g *generator) rpPortalKeyvault() *arm.Resource {
	vault := &mgmtkeyvault.Vault{
		Properties: &mgmtkeyvault.VaultProperties{
			EnableSoftDelete: to.BoolPtr(true),
			TenantID:         &tenantUUIDHack,
			Sku: &mgmtkeyvault.Sku{
				Name:   mgmtkeyvault.Standard,
				Family: to.StringPtr("A"),
			},
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{
				{
					ObjectID: to.StringPtr(portalAccessPolicyHack),
				},
			},
		},
		Name:     to.StringPtr("[concat(parameters('keyvaultPrefix'), '" + env.PortalKeyvaultSuffix + "')]"),
		Type:     to.StringPtr("Microsoft.KeyVault/vaults"),
		Location: to.StringPtr("[resourceGroup().location]"),
	}

	if !g.production {
		*vault.Properties.AccessPolicies = append(g.rpPortalKeyvaultAccessPolicies(),
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
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
	}
}

func (g *generator) rpServiceKeyvault() *arm.Resource {
	vault := &mgmtkeyvault.Vault{
		Properties: &mgmtkeyvault.VaultProperties{
			EnableSoftDelete: to.BoolPtr(true),
			TenantID:         &tenantUUIDHack,
			Sku: &mgmtkeyvault.Sku{
				Name:   mgmtkeyvault.Standard,
				Family: to.StringPtr("A"),
			},
			AccessPolicies: &[]mgmtkeyvault.AccessPolicyEntry{
				{
					ObjectID: to.StringPtr(serviceAccessPolicyHack),
				},
			},
		},
		Name:     to.StringPtr("[concat(parameters('keyvaultPrefix'), '" + env.ServiceKeyvaultSuffix + "')]"),
		Type:     to.StringPtr("Microsoft.KeyVault/vaults"),
		Location: to.StringPtr("[resourceGroup().location]"),
	}

	if !g.production {
		*vault.Properties.AccessPolicies = append(g.rpServiceKeyvaultAccessPolicies(),
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
						mgmtkeyvault.SecretPermissionsGet,
						mgmtkeyvault.SecretPermissionsSet,
						mgmtkeyvault.SecretPermissionsList,
					},
				},
			},
		)
	}

	return &arm.Resource{
		Resource:   vault,
		APIVersion: azureclient.APIVersion("Microsoft.KeyVault"),
	}
}

func (g *generator) rpCosmosDB() []*arm.Resource {
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
			BackupPolicy: mgmtdocumentdb.PeriodicModeBackupPolicy{
				Type: mgmtdocumentdb.TypePeriodic,
				PeriodicModeProperties: &mgmtdocumentdb.PeriodicModeProperties{
					BackupIntervalInMinutes:        to.Int32Ptr(240), //4 hours
					BackupRetentionIntervalInHours: to.Int32Ptr(720), //30 days
				},
			},
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
		APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
	}
	if g.production {
		cosmosdb.IPRules = &[]mgmtdocumentdb.IPAddressOrRange{}
		cosmosdb.IsVirtualNetworkFilterEnabled = to.BoolPtr(true)
		cosmosdb.VirtualNetworkRules = &[]mgmtdocumentdb.VirtualNetworkRule{}
		cosmosdb.DisableKeyBasedMetadataWriteAccess = to.BoolPtr(true)
	}

	rs := []*arm.Resource{
		r,
	}

	if g.production {
		rs = append(rs, g.database("'ARO'", true)...)
	}

	return rs
}

func (g *generator) database(databaseName string, addDependsOn bool) []*arm.Resource {
	portal := &arm.Resource{
		Resource: &mgmtdocumentdb.SQLContainerCreateUpdateParameters{
			SQLContainerCreateUpdateProperties: &mgmtdocumentdb.SQLContainerCreateUpdateProperties{
				Resource: &mgmtdocumentdb.SQLContainerResource{
					ID: to.StringPtr("Portal"),
					PartitionKey: &mgmtdocumentdb.ContainerPartitionKey{
						Paths: &[]string{
							"/id",
						},
						Kind: mgmtdocumentdb.PartitionKindHash,
					},
					DefaultTTL: to.Int32Ptr(-1),
				},
				Options: &mgmtdocumentdb.CreateUpdateOptions{},
			},
			Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Portal')]"),
			Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
		DependsOn: []string{
			"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
		},
	}

	gateway := &arm.Resource{
		Resource: &mgmtdocumentdb.SQLContainerCreateUpdateParameters{
			SQLContainerCreateUpdateProperties: &mgmtdocumentdb.SQLContainerCreateUpdateProperties{
				Resource: &mgmtdocumentdb.SQLContainerResource{
					ID: to.StringPtr("Gateway"),
					PartitionKey: &mgmtdocumentdb.ContainerPartitionKey{
						Paths: &[]string{
							"/id",
						},
						Kind: mgmtdocumentdb.PartitionKindHash,
					},
					DefaultTTL: to.Int32Ptr(-1),
				},
				Options: &mgmtdocumentdb.CreateUpdateOptions{},
			},
			Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Gateway')]"),
			Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
			Location: to.StringPtr("[resourceGroup().location]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
		DependsOn: []string{
			"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
		},
	}

	if g.production {
		portal.Resource.(*mgmtdocumentdb.SQLContainerCreateUpdateParameters).SQLContainerCreateUpdateProperties.Options.Throughput = to.Int32Ptr(400)
		gateway.Resource.(*mgmtdocumentdb.SQLContainerCreateUpdateParameters).SQLContainerCreateUpdateProperties.Options.Throughput = to.Int32Ptr(400)
	}

	rs := []*arm.Resource{
		{
			Resource: &mgmtdocumentdb.SQLDatabaseCreateUpdateParameters{
				SQLDatabaseCreateUpdateProperties: &mgmtdocumentdb.SQLDatabaseCreateUpdateProperties{
					Resource: &mgmtdocumentdb.SQLDatabaseResource{
						ID: to.StringPtr("[" + databaseName + "]"),
					},
					Options: &mgmtdocumentdb.CreateUpdateOptions{
						Throughput: to.Int32Ptr(500),
					},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ")]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
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
					Options: &mgmtdocumentdb.CreateUpdateOptions{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/AsyncOperations')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
		},
		{
			Resource: &mgmtdocumentdb.SQLContainerCreateUpdateParameters{
				SQLContainerCreateUpdateProperties: &mgmtdocumentdb.SQLContainerCreateUpdateProperties{
					Resource: &mgmtdocumentdb.SQLContainerResource{
						ID: to.StringPtr("Billing"),
						PartitionKey: &mgmtdocumentdb.ContainerPartitionKey{
							Paths: &[]string{
								"/id",
							},
							Kind: mgmtdocumentdb.PartitionKindHash,
						},
					},
					Options: &mgmtdocumentdb.CreateUpdateOptions{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Billing')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
		},
		gateway,
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
					Options: &mgmtdocumentdb.CreateUpdateOptions{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Monitors')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
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
					Options: &mgmtdocumentdb.CreateUpdateOptions{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/OpenShiftClusters')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
			DependsOn: []string{
				"[resourceId('Microsoft.DocumentDB/databaseAccounts/sqlDatabases', parameters('databaseAccountName'), " + databaseName + ")]",
			},
		},
		portal,
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
					Options: &mgmtdocumentdb.CreateUpdateOptions{},
				},
				Name:     to.StringPtr("[concat(parameters('databaseAccountName'), '/', " + databaseName + ", '/Subscriptions')]"),
				Type:     to.StringPtr("Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"),
				Location: to.StringPtr("[resourceGroup().location]"),
			},
			APIVersion: azureclient.APIVersion("Microsoft.DocumentDB"),
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

func (g *generator) rpRoleDefinitionTokenContributor() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtauthorization.RoleDefinition{
			Name: to.StringPtr("48983534-3d06-4dcb-a566-08a694eb1279"),
			Type: to.StringPtr("Microsoft.Authorization/roleDefinitions"),
			RoleDefinitionProperties: &mgmtauthorization.RoleDefinitionProperties{
				RoleName:         to.StringPtr("ARO v4 ContainerRegistry Token Contributor"),
				AssignableScopes: &[]string{"[subscription().id]"},
				Permissions: &[]mgmtauthorization.Permission{
					{
						Actions: &[]string{
							"Microsoft.ContainerRegistry/registries/generateCredentials/action",
							"Microsoft.ContainerRegistry/registries/scopeMaps/read",
							"Microsoft.ContainerRegistry/registries/tokens/delete",
							"Microsoft.ContainerRegistry/registries/tokens/operationStatuses/read",
							"Microsoft.ContainerRegistry/registries/tokens/read",
							"Microsoft.ContainerRegistry/registries/tokens/write",
						},
					},
				},
			},
		},
		APIVersion: azureclient.APIVersion("Microsoft.Authorization/roleDefinitions"),
	}
}

func (g *generator) rpRBAC() []*arm.Resource {
	return []*arm.Resource{
		rbac.ResourceGroupRoleAssignmentWithName(
			rbac.RoleReader,
			"parameters('rpServicePrincipalId')",
			"guid(resourceGroup().id, parameters('rpServicePrincipalId'), 'RP / Reader')",
		),
		rbac.ResourceGroupRoleAssignmentWithName(
			rbac.RoleNetworkContributor,
			"parameters('fpServicePrincipalId')",
			"guid(resourceGroup().id, 'FP / Network Contributor')",
		),
		rbac.ResourceRoleAssignmentWithName(
			rbac.RoleDocumentDBAccountContributor,
			"parameters('rpServicePrincipalId')",
			"Microsoft.DocumentDB/databaseAccounts",
			"parameters('databaseAccountName')",
			"concat(parameters('databaseAccountName'), '/Microsoft.Authorization/', guid(resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName')), parameters('rpServicePrincipalId'), 'RP / DocumentDB Account Contributor'))",
		),
		rbac.ResourceRoleAssignmentWithName(
			rbac.RoleDNSZoneContributor,
			"parameters('fpServicePrincipalId')",
			"Microsoft.Network/dnsZones",
			"concat(resourceGroup().location, '.', parameters('clusterParentDomainName'))",
			"concat(resourceGroup().location, '.', parameters('clusterParentDomainName'), '/Microsoft.Authorization/', guid(resourceId('Microsoft.Network/dnsZones', concat(resourceGroup().location, '.', parameters('clusterParentDomainName'))), 'FP / DNS Zone Contributor'))",
		),
	}
}

func (g *generator) rpBillingContributorRbac() []*arm.Resource {
	return []*arm.Resource{
		rbac.ResourceRoleAssignmentWithName(
			rbac.RoleDocumentDBAccountContributor,
			"parameters('billingServicePrincipalId')",
			"Microsoft.DocumentDB/databaseAccounts",
			"parameters('databaseAccountName')",
			"concat(parameters('databaseAccountName'), '/Microsoft.Authorization/', guid(resourceId('Microsoft.DocumentDB/databaseAccounts', parameters('databaseAccountName')), parameters('billingServicePrincipalId') , 'Billing / DocumentDB Account Contributor'))",
			"[greater(length(parameters('billingServicePrincipalId')), 0)]",
		),
	}
}

func (g *generator) rpACR() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtcontainerregistry.Registry{
			Sku: &mgmtcontainerregistry.Sku{
				Name: mgmtcontainerregistry.Premium,
			},
			RegistryProperties: &mgmtcontainerregistry.RegistryProperties{
				// enable data hostname stability: https://azure.microsoft.com/en-gb/blog/azure-container-registry-mitigating-data-exfiltration-with-dedicated-data-endpoints/
				DataEndpointEnabled: to.BoolPtr(true),
			},
			Name: to.StringPtr("[substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1))]"),
			Type: to.StringPtr("Microsoft.ContainerRegistry/registries"),
			// TODO: INT ACR has wrong location - should be redeployed at globalResourceGroupLocation then remove acrLocationOverride configurable.
			Location: to.StringPtr("[if(equals(parameters('acrLocationOverride'), ''), resourceGroup().location, parameters('acrLocationOverride'))]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.ContainerRegistry"),
	}
}

func (g *generator) rpACRReplica() *arm.Resource {
	return &arm.Resource{
		Resource: &mgmtcontainerregistry.Replication{
			Name:     to.StringPtr("[concat(substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1)), '/', parameters('location'))]"),
			Type:     to.StringPtr("Microsoft.ContainerRegistry/registries/replications"),
			Location: to.StringPtr("[parameters('location')]"),
		},
		APIVersion: azureclient.APIVersion("Microsoft.ContainerRegistry"),
	}
}

func (g *generator) rpACRRBAC() []*arm.Resource {
	return []*arm.Resource{
		rbac.ResourceRoleAssignmentWithName(
			rbac.RoleACRPull,
			"parameters('rpServicePrincipalId')",
			"Microsoft.ContainerRegistry/registries",
			"substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1))",
			"concat(substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1)), '/', '/Microsoft.Authorization/', guid(concat(parameters('acrResourceId'), parameters('rpServicePrincipalId'), 'RP / AcrPull')))",
		),
		rbac.ResourceRoleAssignmentWithName(
			rbac.RoleACRPull,
			"parameters('gatewayServicePrincipalId')",
			"Microsoft.ContainerRegistry/registries",
			"substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1))",
			"concat(substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1)), '/', '/Microsoft.Authorization/', guid(concat(parameters('acrResourceId'), parameters('gatewayServicePrincipalId'), 'RP / AcrPull')))",
		),
		rbac.ResourceRoleAssignmentWithName(
			"48983534-3d06-4dcb-a566-08a694eb1279", // ARO v4 ContainerRegistry Token Contributor
			"parameters('fpServicePrincipalId')",
			"Microsoft.ContainerRegistry/registries",
			"substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1))",
			"concat(substring(parameters('acrResourceId'), add(lastIndexOf(parameters('acrResourceId'), '/'), 1)), '/', '/Microsoft.Authorization/', guid(concat(parameters('acrResourceId'), 'FP / ARO v4 ContainerRegistry Token Contributor')))",
		),
	}
}

func (g *generator) rpVersionStorageAccount() []*arm.Resource {
	return []*arm.Resource{
		g.storageAccount("[parameters('rpVersionStorageAccountName')]", &mgmtstorage.AccountProperties{
			AllowBlobPublicAccess: to.BoolPtr(true),
		}),
		{
			Resource: &mgmtstorage.BlobContainer{
				Name: to.StringPtr("[concat(parameters('rpVersionStorageAccountName'), '/default/rpversion')]"),
				Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
				ContainerProperties: &mgmtstorage.ContainerProperties{
					PublicAccess: mgmtstorage.PublicAccessContainer,
				},
			},
			APIVersion: azureclient.APIVersion("Microsoft.Storage"),
			DependsOn: []string{
				"[resourceId('Microsoft.Storage/storageAccounts', parameters('rpVersionStorageAccountName'))]",
			},
		},
		{
			Resource: &mgmtstorage.BlobContainer{
				Name: to.StringPtr("[concat(parameters('rpVersionStorageAccountName'), '/default/ocpversions')]"),
				Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
				ContainerProperties: &mgmtstorage.ContainerProperties{
					PublicAccess: mgmtstorage.PublicAccessContainer,
				},
			},
			APIVersion: azureclient.APIVersion("Microsoft.Storage"),
			DependsOn: []string{
				"[resourceId('Microsoft.Storage/storageAccounts', parameters('rpVersionStorageAccountName'))]",
			},
		},
	}
}

func (g *generator) rpStorageAccount() *arm.Resource {
	return g.storageAccount("[substring(parameters('storageAccountDomain'), 0, indexOf(parameters('storageAccountDomain'), '.'))]", nil)
}
