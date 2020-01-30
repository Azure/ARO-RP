package template

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/resources"
)

type storageTemplate struct {
	log            *logrus.Entry
	template       *arm.Template
	location       string
	storageSuffix  string
	resourceGroup  string
	subscriptionID string
	deployments    resources.DeploymentsClient
}

func NewStorageTemplate(log *logrus.Entry, subscriptionID, location, resourceGroup string, deployments resources.DeploymentsClient, storageSuffix string) Template {
	return &storageTemplate{
		log:            log,
		resourceGroup:  resourceGroup,
		subscriptionID: subscriptionID,
		deployments:    deployments,
		location:       location,
		storageSuffix:  storageSuffix,
	}
}

func (s *storageTemplate) Deploy(ctx context.Context) error {
	return templateDeploy(ctx, s.log, s.deployments, s.Generate(), nil, s.resourceGroup)
}

func (s *storageTemplate) Generate() *arm.Template {
	return &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources: []*arm.Resource{
			{
				Resource: &mgmtstorage.Account{
					Sku: &mgmtstorage.Sku{
						Name: "Standard_LRS",
					},
					Name:     to.StringPtr("cluster" + s.storageSuffix),
					Location: &s.location,
					Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
				},
				APIVersion: apiVersions["storage"],
			},
			{
				Resource: &mgmtstorage.BlobContainer{
					Name: to.StringPtr("cluster" + s.storageSuffix + "/default/ignition"),
					Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
				},
				APIVersion: apiVersions["storage"],
				DependsOn: []string{
					"Microsoft.Storage/storageAccounts/cluster" + s.storageSuffix,
				},
			},
			{
				Resource: &mgmtstorage.BlobContainer{
					Name: to.StringPtr("cluster" + s.storageSuffix + "/default/aro"),
					Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
				},
				APIVersion: apiVersions["storage"],
				DependsOn: []string{
					"Microsoft.Storage/storageAccounts/cluster" + s.storageSuffix,
				},
			},
			{
				Resource: &mgmtnetwork.SecurityGroup{
					SecurityGroupPropertiesFormat: &mgmtnetwork.SecurityGroupPropertiesFormat{
						SecurityRules: &[]mgmtnetwork.SecurityRule{
							{
								SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
									Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
									SourcePortRange:          to.StringPtr("*"),
									DestinationPortRange:     to.StringPtr("6443"),
									SourceAddressPrefix:      to.StringPtr("*"),
									DestinationAddressPrefix: to.StringPtr("*"),
									Access:                   mgmtnetwork.SecurityRuleAccessAllow,
									Priority:                 to.Int32Ptr(101),
									Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
								},
								Name: to.StringPtr("apiserver_in"),
							},
							{
								SecurityRulePropertiesFormat: &mgmtnetwork.SecurityRulePropertiesFormat{
									Protocol:                 mgmtnetwork.SecurityRuleProtocolTCP,
									SourcePortRange:          to.StringPtr("*"),
									DestinationPortRange:     to.StringPtr("22"),
									SourceAddressPrefix:      to.StringPtr("*"),
									DestinationAddressPrefix: to.StringPtr("*"),
									Access:                   mgmtnetwork.SecurityRuleAccessAllow,
									Priority:                 to.Int32Ptr(103),
									Direction:                mgmtnetwork.SecurityRuleDirectionInbound,
								},
								Name: to.StringPtr("bootstrap_ssh_in"),
							},
						},
					},
					Name:     to.StringPtr("aro-controlplane-nsg"),
					Type:     to.StringPtr("Microsoft.Network/networkSecurityGroups"),
					Location: &s.location,
				},
				APIVersion: apiVersions["network"],
			},
			{
				Resource: &mgmtnetwork.SecurityGroup{
					Name:     to.StringPtr("aro-node-nsg"),
					Type:     to.StringPtr("Microsoft.Network/networkSecurityGroups"),
					Location: &s.location,
				},
				APIVersion: apiVersions["network"],
			},
		},
	}
}
