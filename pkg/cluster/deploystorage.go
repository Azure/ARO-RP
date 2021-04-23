package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-07-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/installer/pkg/asset/installconfig"
	"github.com/openshift/installer/pkg/asset/kubeconfig"
	"github.com/openshift/installer/pkg/asset/releaseimage"
	"github.com/openshift/installer/pkg/asset/targets"
	"github.com/openshift/installer/pkg/asset/templates/content/bootkube"
	"github.com/openshift/installer/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/bootstraplogging"
	"github.com/Azure/ARO-RP/pkg/cluster/graph"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func (m *manager) createDNS(ctx context.Context) error {
	return m.dns.Create(ctx, m.doc.OpenShiftCluster)
}

func (m *manager) ensureInfraID(ctx context.Context, installConfig *installconfig.InstallConfig) error {
	if m.doc.OpenShiftCluster.Properties.InfraID != "" {
		return nil
	}

	g := graph.Graph{}
	g.Set(&installconfig.InstallConfig{
		Config: &types.InstallConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: strings.ToLower(m.doc.OpenShiftCluster.Name),
			},
		},
	})

	err := g.Resolve(&installconfig.ClusterID{})
	if err != nil {
		return err
	}

	clusterID := g.Get(&installconfig.ClusterID{}).(*installconfig.ClusterID)

	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.InfraID = clusterID.InfraID
		return nil
	})
	return err
}

func (m *manager) ensureResourceGroup(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	group := mgmtfeatures.ResourceGroup{
		Location:  &m.doc.OpenShiftCluster.Location,
		ManagedBy: to.StringPtr(m.doc.OpenShiftCluster.ID),
	}
	if m.env.IsLocalDevelopmentMode() {
		// TODO: ideally this should be removed, but removing it causes the
		// purge script to ignore RGs in CI E2E.  Fix that at the same time.
		group.ManagedBy = nil
	}

	_, err := m.resourceGroups.CreateOrUpdate(ctx, resourceGroup, group)

	var serviceError *azure.ServiceError
	// CreateOrUpdate wraps DetailedError wrapping a *RequestError (if error generated in ResourceGroup CreateOrUpdateResponder at least)
	if detailedErr, ok := err.(autorest.DetailedError); ok {
		if requestErr, ok := detailedErr.Original.(*azure.RequestError); ok {
			serviceError = requestErr.ServiceError
		}
	}

	// TODO [gv]: Keeping this for retro-compatibility, but probably this can be removed
	if requestErr, ok := err.(*azure.RequestError); ok {
		serviceError = requestErr.ServiceError
	}

	if serviceError != nil && serviceError.Code == "RequestDisallowedByPolicy" {
		// if request was disallowed by policy, inform user so they can take appropriate action
		b, _ := json.Marshal(serviceError)
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

	return m.env.EnsureARMResourceGroupRoleAssignment(ctx, m.fpAuthorizer, resourceGroup)
}

func (m *manager) deployStorageTemplate(ctx context.Context, installConfig *installconfig.InstallConfig) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	infraID := m.doc.OpenShiftCluster.Properties.InfraID

	resources := []*arm.Resource{
		m.clusterStorageAccount(installConfig.Config.Azure.Region),
		m.clusterStorageAccountBlob("ignition"),
		m.clusterStorageAccountBlob("aro"),
		m.clusterNSG(infraID, installConfig.Config.Azure.Region),
		m.clusterServicePrincipalRBAC(),
		m.networkPrivateLinkService(installConfig),
		m.networkPublicIPAddress(installConfig, m.doc.OpenShiftCluster.Properties.InfraID+"-pip-v4"),
		m.networkInternalLoadBalancer(installConfig),
		m.networkPublicLoadBalancer(installConfig),
	}

	if m.doc.OpenShiftCluster.Properties.IngressProfiles[0].Visibility == api.VisibilityPublic {
		resources = append(resources,
			m.networkPublicIPAddress(installConfig, m.doc.OpenShiftCluster.Properties.InfraID+"-default-v4"),
		)
	}

	t := &arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources:      resources,
	}

	if !m.env.FeatureIsSet(env.FeatureDisableDenyAssignments) {
		t.Resources = append(t.Resources, m.denyAssignment())
	}

	return m.deployARMTemplate(ctx, resourceGroup, "storage", t, nil)
}

func (m *manager) ensureGraph(ctx context.Context, installConfig *installconfig.InstallConfig, image *releaseimage.Image) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	account := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix

	exists, err := m.graph.Exists(ctx, resourceGroup, account)
	if err != nil || exists {
		return err
	}

	infraID := m.doc.OpenShiftCluster.Properties.InfraID

	clusterID := &installconfig.ClusterID{
		UUID:    m.doc.ID,
		InfraID: infraID,
	}

	bootstrapLoggingConfig, err := bootstraplogging.GetConfig(m.env, m.doc)
	if err != nil {
		return err
	}

	dnsConfig := &bootkube.ARODNSConfig{
		APIIntIP:  m.doc.OpenShiftCluster.Properties.APIServerProfile.IntIP,
		IngressIP: m.doc.OpenShiftCluster.Properties.IngressProfiles[0].IP,
	}

	g := graph.Graph{}
	g.Set(installConfig, image, clusterID, bootstrapLoggingConfig, dnsConfig)

	m.log.Print("resolving graph")
	for _, a := range targets.Cluster {
		err = g.Resolve(a)
		if err != nil {
			return err
		}
	}

	// the graph is quite big so we store it in a storage account instead of in cosmosdb
	return m.graph.Save(ctx, resourceGroup, account, g)
}

func (m *manager) attachNSGsAndPatch(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	account := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix

	pg, err := m.graph.LoadPersisted(ctx, resourceGroup, account)
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

	var adminInternalClient *kubeconfig.AdminInternalClient
	err = pg.Get(&adminInternalClient)
	if err != nil {
		return err
	}

	aroServiceInternalClient, err := m.generateAROServiceKubeconfig(pg)
	if err != nil {
		return err
	}
	aroSREInternalClient, err := m.generateAROSREKubeconfig(pg)
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
