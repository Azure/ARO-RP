package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
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
	"github.com/openshift/installer/pkg/asset/bootstraplogging"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/releaseimage"
	"github.com/openshift/installer/pkg/asset/targets"
	"github.com/openshift/installer/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/feature"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func (m *manager) createDNS(ctx context.Context) error {
	return m.dns.Create(ctx, m.doc.OpenShiftCluster)
}

func (m *manager) clusterSPObjectID(ctx context.Context) (string, error) {
	var clusterSPObjectID string
	spp := &m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile

	token, err := aad.GetToken(ctx, m.log, m.doc.OpenShiftCluster, azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return "", err
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
				m.log.Info(err)
				return false, nil
			}
			return false, err
		}

		clusterSPObjectID = *res.Value
		return true, nil
	}, timeoutCtx.Done())

	return clusterSPObjectID, err
}

func (m *manager) deployStorageTemplate(ctx context.Context, installConfig *installconfig.InstallConfig, platformCreds *installconfig.PlatformCreds, image *releaseimage.Image, bootstrapLoggingConfig *bootstraplogging.Config) error {
	if m.doc.OpenShiftCluster.Properties.InfraID == "" {
		clusterID := &installconfig.ClusterID{}

		err := clusterID.Generate(asset.Parents{
			reflect.TypeOf(installConfig): &installconfig.InstallConfig{
				Config: &types.InstallConfig{
					ObjectMeta: metav1.ObjectMeta{
						Name: strings.ToLower(m.doc.OpenShiftCluster.Name),
					},
				},
			},
		})
		if err != nil {
			return err
		}

		m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
			doc.OpenShiftCluster.Properties.InfraID = clusterID.InfraID
			return nil
		})
		if err != nil {
			return err
		}
	}
	infraID := m.doc.OpenShiftCluster.Properties.InfraID

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	m.log.Print("creating resource group")
	group := mgmtfeatures.ResourceGroup{
		Location:  &installConfig.Config.Azure.Region,
		ManagedBy: to.StringPtr(m.doc.OpenShiftCluster.ID),
	}
	if m.env.DeploymentMode() == deployment.Development {
		group.ManagedBy = nil
	}
	_, err := m.groups.CreateOrUpdate(ctx, resourceGroup, group)
	if requestErr, ok := err.(*azure.RequestError); ok &&
		requestErr.ServiceError != nil && requestErr.ServiceError.Code == "RequestDisallowedByPolicy" {
		// if request was disallowed by policy, inform user so they can take appropriate action
		b, _ := json.Marshal(requestErr.ServiceError)
		return &api.CloudError{
			StatusCode: http.StatusBadRequest,
			CloudErrorBody: &api.CloudErrorBody{
				Code:    api.CloudErrorCodeDeploymentFailed,
				Message: "Deployment failed.",
				Details: []api.CloudErrorBody{
					{
						Message: string(b),
					},
				},
			},
		}
	}
	if err != nil {
		return err
	}

	err = m.env.CreateARMResourceGroupRoleAssignment(ctx, m.fpAuthorizer, resourceGroup)
	if err != nil {
		return err
	}

	clusterSPObjectID, err := m.clusterSPObjectID(ctx)
	if err != nil {
		return err
	}

	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources: []*arm.Resource{
			rbac.ResourceGroupRoleAssignmentWithName(
				rbac.RoleContributor,
				"'"+clusterSPObjectID+"'",
				"guid(resourceGroup().id, 'SP / Contributor')",
			),
			{
				Resource: &mgmtstorage.Account{
					Sku: &mgmtstorage.Sku{
						Name: "Standard_LRS",
					},
					Name:     to.StringPtr("cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix),
					Location: &installConfig.Config.Azure.Region,
					Type:     to.StringPtr("Microsoft.Storage/storageAccounts"),
				},
				APIVersion: azureclient.APIVersions["Microsoft.Storage"],
			},
			{
				Resource: &mgmtstorage.BlobContainer{
					Name: to.StringPtr("cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix + "/default/ignition"),
					Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
				},
				APIVersion: azureclient.APIVersions["Microsoft.Storage"],
				DependsOn: []string{
					"Microsoft.Storage/storageAccounts/cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix,
				},
			},
			{
				Resource: &mgmtstorage.BlobContainer{
					Name: to.StringPtr("cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix + "/default/aro"),
					Type: to.StringPtr("Microsoft.Storage/storageAccounts/blobServices/containers"),
				},
				APIVersion: azureclient.APIVersions["Microsoft.Storage"],
				DependsOn: []string{
					"Microsoft.Storage/storageAccounts/cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix,
				},
			},
			m.clusterNSG(installConfig.Config.Azure.Region),
		},
	}

	if m.env.DeploymentMode() == deployment.Production {
		t.Resources = append(t.Resources, m.denyAssignments(clusterSPObjectID))
	}

	err = m.deployARMTemplate(ctx, resourceGroup, "storage", t, nil)
	if err != nil {
		return err
	}

	exists, err := m.graphExists(ctx)
	if err != nil || exists {
		return err
	}

	clusterID := &installconfig.ClusterID{
		UUID:    m.doc.ID,
		InfraID: infraID,
	}

	g := graph{
		reflect.TypeOf(installConfig):          installConfig,
		reflect.TypeOf(platformCreds):          platformCreds,
		reflect.TypeOf(image):                  image,
		reflect.TypeOf(clusterID):              clusterID,
		reflect.TypeOf(bootstrapLoggingConfig): bootstrapLoggingConfig,
	}

	m.log.Print("resolving graph")
	for _, a := range targets.Cluster {
		_, err := g.resolve(a)
		if err != nil {
			return err
		}
	}

	// the graph is quite big so we store it in a storage account instead of in cosmosdb
	return m.saveGraph(ctx, g)
}

var extraDenyAssignmentExclusions = map[string][]string{
	"Microsoft.RedHatOpenShift/RedHatEngineering": {
		"Microsoft.Network/networkInterfaces/effectiveRouteTable/action",
		"Microsoft.Resources/resourceGroups/write", // enable resource group tagging
	},
}

func (m *manager) denyAssignments(clusterSPObjectID string) *arm.Resource {
	notActions := []string{
		"Microsoft.Network/networkSecurityGroups/join/action",
		"Microsoft.Compute/disks/beginGetAccess/action",
		"Microsoft.Compute/disks/endGetAccess/action",
		"Microsoft.Compute/disks/write",
		"Microsoft.Compute/snapshots/beginGetAccess/action",
		"Microsoft.Compute/snapshots/endGetAccess/action",
		"Microsoft.Compute/snapshots/write",
		"Microsoft.Compute/snapshots/delete",
	}

	var props = m.subscriptionDoc.Subscription.Properties

	for flag, exclusions := range extraDenyAssignmentExclusions {
		if feature.IsRegisteredForFeature(props, flag) {
			notActions = append(notActions, exclusions...)
		}
	}

	return &arm.Resource{
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
						NotActions: &notActions,
					},
				},
				Scope: &m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID,
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
	}
}

func (m *manager) deploySnapshotUpgradeTemplate(ctx context.Context) error {
	if m.env.DeploymentMode() != deployment.Production {
		// only need this upgrade in production, where there are DenyAssignments
		return nil
	}

	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	clusterSPObjectID, err := m.clusterSPObjectID(ctx)
	if err != nil {
		return err
	}

	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources:      []*arm.Resource{m.denyAssignments(clusterSPObjectID)},
	}

	return m.deployARMTemplate(ctx, resourceGroup, "storage", t, nil)
}

func (m *manager) attachNSGsAndPatch(ctx context.Context) error {
	g, err := m.loadGraph(ctx)
	if err != nil {
		return err
	}

	for _, subnetID := range []string{
		m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID,
		m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].SubnetID,
	} {
		m.log.Printf("attaching network security group to subnet %s", subnetID)

		// TODO: there is probably an undesirable race condition here - check if etags can help.

		s, err := m.subnet.Get(ctx, subnetID)
		if err != nil {
			return err
		}

		if s.SubnetPropertiesFormat == nil {
			s.SubnetPropertiesFormat = &mgmtnetwork.SubnetPropertiesFormat{}
		}

		nsgID, err := subnet.NetworkSecurityGroupID(m.doc.OpenShiftCluster, subnetID)
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

		err = m.subnet.CreateOrUpdate(ctx, subnetID, s)
		if err != nil {
			return err
		}
	}

	adminInternalClient := g[reflect.TypeOf(&kubeconfig.AdminInternalClient{})].(*kubeconfig.AdminInternalClient)
	aroServiceInternalClient, err := m.generateAROServiceKubeconfig(g)
	if err != nil {
		return err
	}

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
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
