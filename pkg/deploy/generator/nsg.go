package generator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"io/ioutil"

	mgmtmsi "github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/ARO-RP/pkg/util/arm"
)

func GenerateNSGTemplates() error {
	for _, i := range []struct {
		templateFile string
		production   bool
	}{
		{
			templateFile: fileRPDevelopmentNSG,
		},
		{
			templateFile: FileRPProductionNSG,
			production:   true,
		},
	} {
		nsg := &mgmtnetwork.SecurityGroup{
			SecurityGroupPropertiesFormat: &mgmtnetwork.SecurityGroupPropertiesFormat{
				SecurityRules: &[]mgmtnetwork.SecurityRule{
					{
						SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
							Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
							SourcePortRange:          to.StringPtr("*"),
							DestinationPortRange:     to.StringPtr("443"),
							SourceAddressPrefix:      to.StringPtr("*"),
							DestinationAddressPrefix: to.StringPtr("*"),
							Access:                   mgmtnetwork.SecurityRuleAccessAllow,
							Priority:                 to.Int32Ptr(120),
							Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
						},
						Name: to.StringPtr("rp_in"),
					},
				},
			},
			Name:     to.StringPtr("rp-nsg"),
			Type:     to.StringPtr("Microsoft.Network/networkSecurityGroups"),
			Location: to.StringPtr("[resourceGroup().location]"),
		}

		if !i.production {
			*nsg.SecurityRules = append(*nsg.SecurityRules, mgmtnetwork.SecurityRule{
				SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
					Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
					SourcePortRange:          to.StringPtr("*"),
					DestinationPortRange:     to.StringPtr("22"),
					SourceAddressPrefix:      to.StringPtr("*"),
					DestinationAddressPrefix: to.StringPtr("*"),
					Access:                   mgmtnetwork.SecurityRuleAccessAllow,
					Priority:                 to.Int32Ptr(100),
					Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
				},
				Name: to.StringPtr("ssh_in"),
			})
		}

		t := &arm.Template{
			Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
			ContentVersion: "1.0.0.0",
			Resources: []*arm.Resource{
				{
					Resource:   nsg,
					APIVersion: apiVersions["network"],
				},
				{
					Resource: &mgmtnetwork.SecurityGroup{
						SecurityGroupPropertiesFormat: &mgmtnetwork.SecurityGroupPropertiesFormat{},
						Name:                          to.StringPtr("rp-pe-nsg"),
						Type:                          to.StringPtr("Microsoft.Network/networkSecurityGroups"),
						Location:                      to.StringPtr("[resourceGroup().location]"),
					},
					APIVersion: apiVersions["network"],
				},
			},
		}

		if i.production {
			t.Resources = append(t.Resources,
				&arm.Resource{
					Resource: &mgmtmsi.Identity{
						Name:     to.StringPtr("rp-identity"),
						Location: to.StringPtr("[resourceGroup().location]"),
						Type:     "Microsoft.ManagedIdentity/userAssignedIdentities",
					},
					APIVersion: apiVersions["msi"],
				},
			)

			t.Outputs = map[string]*arm.Output{
				"rpServicePrincipalId": {
					Type:  "string",
					Value: "[reference(resourceId('Microsoft.ManagedIdentity/userAssignedIdentities', 'rp-identity'), '2018-11-30').principalId]",
				},
			}
		}

		b, err := json.MarshalIndent(t, "", "    ")
		if err != nil {
			return err
		}

		b = append(b, byte('\n'))

		err = ioutil.WriteFile(i.templateFile, b, 0666)
		if err != nil {
			return err
		}
	}

	return nil
}
