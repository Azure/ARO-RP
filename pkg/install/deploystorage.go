package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	azgraphrbac "github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtauthorization "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-04-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/releaseimage"
	"github.com/openshift/installer/pkg/asset/targets"
	"github.com/openshift/installer/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func (i *Installer) createDNS(ctx context.Context) error {
	return i.dns.Create(ctx, i.doc.OpenShiftCluster)
}

func (i *Installer) deployStorageTemplate(ctx context.Context, installConfig *installconfig.InstallConfig, platformCreds *installconfig.PlatformCreds, image *releaseimage.Image) error {
	if i.doc.OpenShiftCluster.Properties.InfraID == "" {
		clusterID := &installconfig.ClusterID{}

		err := clusterID.Generate(asset.Parents{
			reflect.TypeOf(installConfig): &installconfig.InstallConfig{
				Config: &types.InstallConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name: strings.ToLower(i.doc.OpenShiftCluster.Name),
					},
				},
			},
		})
		if err != nil {
			return err
		}

		i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
			doc.OpenShiftCluster.Properties.InfraID = clusterID.InfraID
			return nil
		})
		if err != nil {
			return err
		}
	}
	infraID := i.doc.OpenShiftCluster.Properties.InfraID

	resourceGroup := stringutils.LastTokenByte(i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	i.log.Print("creating resource group")
	group := mgmtfeatures.ResourceGroup{
		Location:  &installConfig.Config.Azure.Region,
		ManagedBy: to.StringPtr(i.doc.OpenShiftCluster.ID),
	}
	if _, ok := i.env.(env.Dev); ok {
		group.ManagedBy = nil
	}
	_, err := i.groups.CreateOrUpdate(ctx, resourceGroup, group)
	if err != nil {
		return err
	}

	if development, ok := i.env.(env.Dev); ok {
		err = development.CreateARMResourceGroupRoleAssignment(ctx, i.fpAuthorizer, resourceGroup)
		if err != nil {
			return err
		}
	}

	var clusterSPObjectID string
	{
		spp := &i.doc.OpenShiftCluster.Properties.ServicePrincipalProfile

		token, err := aad.GetToken(ctx, i.log, &i.doc.OpenShiftCluster.Properties.ServicePrincipalProfile, azure.PublicCloud.GraphEndpoint)
		if err != nil {
			return err
		}

		spGraphAuthorizer := autorest.NewBearerAuthorizer(token)

		applications := graphrbac.NewApplicationsClient(spp.TenantID, spGraphAuthorizer)

		timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		// NOTE: Do not override err with the error returned by wait.PollImmediateUntil.
		// Doing this will not propagate the latest error to the user in case when wait exceeds the timeout
		wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
			var res azgraphrbac.ServicePrincipalObjectResult
			res, err = applications.GetServicePrincipalsIDByAppID(ctx, spp.ClientID)
			if err != nil {
				if strings.Contains(err.Error(), "Authorization_IdentityNotFound") {
					i.log.Info(err)
					return false, nil
				}

				return false, err
			}

			clusterSPObjectID = *res.Value
			return true, nil
		}, timeoutCtx.Done())
		if err != nil {
			return err
		}
	}

	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources: []*arm.Resource{
			{
				Resource: &mgmtauthorization.RoleAssignment{
					Name: to.StringPtr("[guid(resourceGroup().id, 'SP / Contributor')]"),
					Type: to.StringPtr("Microsoft.Authorization/roleAssignments"),
					RoleAssignmentPropertiesWithScope: &mgmtauthorization.RoleAssignmentPropertiesWithScope{
						Scope:            to.StringPtr("[resourceGroup().id]"),
						RoleDefinitionID: to.StringPtr("[resourceId('Microsoft.Authorization/roleDefinitions', 'b24988ac-6180-42a0-ab88-20f7382dd24c')]"), // Contributor
						PrincipalID:      &clusterSPObjectID,
						PrincipalType:    mgmtauthorization.ServicePrincipal,
					},
				},
				APIVersion: azureclient.APIVersions["Microsoft.Authorization"],
			},
			{
				Resource: &mgmtstorage.Account{
					Sku: &mgmtstorage.Sku{
						Name: "Standard_LRS",
					},
					Name:     to.StringPtr("cluster" + i.doc.OpenShiftCluster.Properties.StorageSuffix),
					Location: &installConfig.Config.Azure.Region,
					Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
				},
				APIVersion: azureclient.APIVersions["Microsoft.Storage"],
			},
			{
				Resource: &mgmtstorage.BlobContainer{
					Name: to.StringPtr("cluster" + i.doc.OpenShiftCluster.Properties.StorageSuffix + "/default/ignition"),
					Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
				},
				APIVersion: azureclient.APIVersions["Microsoft.Storage"],
				DependsOn: []string{
					"Microsoft.Storage/storageAccounts/cluster" + i.doc.OpenShiftCluster.Properties.StorageSuffix,
				},
			},
			{
				Resource: &mgmtstorage.BlobContainer{
					Name: to.StringPtr("cluster" + i.doc.OpenShiftCluster.Properties.StorageSuffix + "/default/aro"),
					Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
				},
				APIVersion: azureclient.APIVersions["Microsoft.Storage"],
				DependsOn: []string{
					"Microsoft.Storage/storageAccounts/cluster" + i.doc.OpenShiftCluster.Properties.StorageSuffix,
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
						},
					},
					Name:     to.StringPtr(infraID + subnet.NSGControlPlaneSuffix),
					Type:     to.StringPtr("Microsoft.Network/networkSecurityGroups"),
					Location: &installConfig.Config.Azure.Region,
				},
				APIVersion: azureclient.APIVersions["Microsoft.Network"],
			},
			{
				Resource: &mgmtnetwork.SecurityGroup{
					Name:     to.StringPtr(infraID + subnet.NSGNodeSuffix),
					Type:     to.StringPtr("Microsoft.Network/networkSecurityGroups"),
					Location: &installConfig.Config.Azure.Region,
				},
				APIVersion: azureclient.APIVersions["Microsoft.Network"],
			},
		},
	}

	if os.Getenv("RP_MODE") == "" {
		t.Resources = append(t.Resources, &arm.Resource{
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
								"Microsoft.Network/networkSecurityGroups/join/action",
							},
						},
					},
					Scope: &i.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID,
					Principals: &[]mgmtauthorization.Principal{
						{
							ID:   to.StringPtr("00000000-0000-0000-0000-000000000000"),
							Type: to.StringPtr("SystemDefined"),
						},
					},
					ExcludePrincipals: &[]mgmtauthorization.Principal{
						{
							ID:   &clusterSPObjectID,
							Type: to.StringPtr("ServicePrincipal"),
						},
					},
					IsSystemProtected: to.BoolPtr(true),
				},
			},
			APIVersion: azureclient.APIVersions["Microsoft.Authorization/denyAssignments"],
		})
	}

	err = i.deployARMTemplate(ctx, resourceGroup, "storage", t, nil)
	if err != nil {
		return err
	}

	exists, err := i.graphExists(ctx)
	if err != nil || exists {
		return err
	}

	clusterID := &installconfig.ClusterID{
		UUID:    i.doc.ID,
		InfraID: infraID,
	}

	g := graph{
		reflect.TypeOf(installConfig): installConfig,
		reflect.TypeOf(platformCreds): platformCreds,
		reflect.TypeOf(image):         image,
		reflect.TypeOf(clusterID):     clusterID,
	}

	i.log.Print("resolving graph")
	for _, a := range targets.Cluster {
		_, err := g.resolve(a)
		if err != nil {
			return err
		}
	}

	// the graph is quite big so we store it in a storage account instead of in cosmosdb
	return i.saveGraph(ctx, g)
}

func (i *Installer) attachNSGsAndPatch(ctx context.Context) error {
	g, err := i.loadGraph(ctx)
	if err != nil {
		return err
	}

	for _, subnetID := range []string{
		i.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID,
		i.doc.OpenShiftCluster.Properties.WorkerProfiles[0].SubnetID,
	} {
		i.log.Printf("attaching network security group to subnet %s", subnetID)

		// TODO: there is probably an undesirable race condition here - check if etags can help.

		s, err := i.subnet.Get(ctx, subnetID)
		if err != nil {
			return err
		}

		if s.SubnetPropertiesFormat == nil {
			s.SubnetPropertiesFormat = &mgmtnetwork.SubnetPropertiesFormat{}
		}

		nsgID, err := subnet.NetworkSecurityGroupID(i.doc.OpenShiftCluster, subnetID)
		if err != nil {
			return err
		}

		// Sometimes we get into the race condition between external services modifying
		// subnets and our validation code. We try to catch this early, but
		// these errors is propagated to make the user-facing error more clear incase
		// modification happened after we ran validation code and we lost the race
		if s.SubnetPropertiesFormat.NetworkSecurityGroup != nil {
			if strings.EqualFold(*s.SubnetPropertiesFormat.NetworkSecurityGroup.ID, nsgID) {
				continue
			}

			return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidLinkedVNet, "", "The provided subnet '%s' is invalid: must not have a network security group attached.", subnetID)
		}

		s.SubnetPropertiesFormat.NetworkSecurityGroup = &mgmtnetwork.SecurityGroup{
			ID: to.StringPtr(nsgID),
		}

		err = i.subnet.CreateOrUpdate(ctx, subnetID, s)
		if err != nil {
			return err
		}
	}

	adminInternalClient := g[reflect.TypeOf(&kubeconfig.AdminInternalClient{})].(*kubeconfig.AdminInternalClient)
	aroServiceInternalClient, err := i.generateAROServiceKubeconfig(g)
	if err != nil {
		return err
	}

	i.doc, err = i.db.PatchWithLease(ctx, i.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		// used for the SAS token with which the bootstrap node retrieves its
		// ignition payload
		var t time.Time
		if doc.OpenShiftCluster.Properties.Install.Now == t {
			// Only set this if it hasn't been set already, since it is used to
			// create values for signedStart and signedExpiry in
			// deployResourceTemplate, and if these are not stable a
			// redeployment will fail.
			doc.OpenShiftCluster.Properties.Install.Now = time.Now().UTC()
		}
		doc.OpenShiftCluster.Properties.AdminKubeconfig = adminInternalClient.File.Data
		doc.OpenShiftCluster.Properties.AROServiceKubeconfig = aroServiceInternalClient.File.Data
		return nil
	})
	return err
}
