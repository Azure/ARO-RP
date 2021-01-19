package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	azgraphrbac "github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/releaseimage"
	"github.com/openshift/installer/pkg/asset/targets"
	"github.com/openshift/installer/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/bootstraplogging"
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/graphrbac"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func (m *manager) createDNS(ctx context.Context) error {
	return m.dns.Create(ctx, m.doc.OpenShiftCluster)
}

func (m *manager) clusterSPObjectID(ctx context.Context) (string, error) {
	var clusterSPObjectID string
	spp := &m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile

	token, err := aad.GetToken(ctx, m.log, m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientID, m.doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret, m.subscriptionDoc.Subscription.Properties.TenantID, m.env.Environment().ActiveDirectoryEndpoint, m.env.Environment().GraphEndpoint)
	if err != nil {
		return "", err
	}

	spGraphAuthorizer := autorest.NewBearerAuthorizer(token)

	applications := graphrbac.NewApplicationsClient(m.env.Environment(), m.subscriptionDoc.Subscription.Properties.TenantID, spGraphAuthorizer)

	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// NOTE: Do not override err with the error returned by
	// wait.PollImmediateUntil. Doing this will not propagate the latest error
	// to the user in case when wait exceeds the timeout
	_ = wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
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

func (m *manager) deployStorageTemplate(ctx context.Context, installConfig *installconfig.InstallConfig, image *releaseimage.Image) error {
	if m.doc.OpenShiftCluster.Properties.InfraID == "" {
		g := newGraph(&installconfig.InstallConfig{
			Config: &types.InstallConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name: strings.ToLower(m.doc.OpenShiftCluster.Name),
				},
			},
		})

		err := g.resolve(&installconfig.ClusterID{})
		if err != nil {
			return err
		}

		clusterID := g.get(&installconfig.ClusterID{}).(*installconfig.ClusterID)

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
	_, err := m.resourceGroups.CreateOrUpdate(ctx, resourceGroup, group)
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
			m.clusterStorageAccount(installConfig.Config.Azure.Region),
			m.clusterStorageAccountBlob("ignition"),
			m.clusterStorageAccountBlob("aro"),
			m.clusterNSG(infraID, installConfig.Config.Azure.Region),
			m.clusterServicePrincipalRBAC(clusterSPObjectID),
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

	bootstrapLoggingConfig, err := bootstraplogging.GetConfig(m.env, m.doc)
	if err != nil {
		return err
	}

	g := newGraph(installConfig, image, clusterID, bootstrapLoggingConfig)

	m.log.Print("resolving graph")
	for _, a := range targets.Cluster {
		err = g.resolve(a)
		if err != nil {
			return err
		}
	}

	// the graph is quite big so we store it in a storage account instead of in cosmosdb
	return m.saveGraph(ctx, g)
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

	adminInternalClient := g.get(&kubeconfig.AdminInternalClient{}).(*kubeconfig.AdminInternalClient)
	aroServiceInternalClient, err := m.generateAROServiceKubeconfig(g)
	if err != nil {
		return err
	}
	aroSREInternalClient, err := m.generateAROSREKubeconfig(g)
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
		doc.OpenShiftCluster.Properties.AROSREKubeconfig = aroSREInternalClient.File.Data
		return nil
	})
	return err
}
