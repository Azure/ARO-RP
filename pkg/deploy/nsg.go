package deploy

import (
	"encoding/json"
	"io/ioutil"

	"github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/jim-minter/rp/pkg/util/arm"
)

func GenerateNSGTemplate() error {
	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources: []*arm.Resource{
			{
				Resource: &msi.Identity{
					Name:     to.StringPtr("rp-identity"),
					Location: to.StringPtr("[resourceGroup().location]"),
					Type:     "Microsoft.ManagedIdentity/userAssignedIdentities",
				},
				APIVersion: apiVersions["msi"],
			},
			{
				Resource: &network.SecurityGroup{
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
									Priority:                 to.Int32Ptr(120),
									Direction:                network.SecurityRuleDirectionInbound,
								},
								Name: to.StringPtr("rp_in"),
							},
						},
					},
					Name:     to.StringPtr("rp-nsg"),
					Type:     to.StringPtr("Microsoft.Network/networkSecurityGroups"),
					Location: to.StringPtr("[resourceGroup().location]"),
				},
				APIVersion: apiVersions["network"],
			},
		},
		Outputs: map[string]interface{}{
			"rpServicePrincipalId": "[reference(resourceId('Microsoft.ManagedIdentity/userAssignedIdentities', 'rp-identity'), '2018-11-30').principalId]",
		},
	}

	b, err := json.MarshalIndent(t, "", "    ")
	if err != nil {
		return err
	}

	b = append(b, byte('\n'))

	return ioutil.WriteFile("rp-production-nsg.json", b, 0666)
}
