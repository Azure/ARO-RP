package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	utilrand "k8s.io/apimachinery/pkg/util/rand"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/arm"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func (m *manager) createDNS(ctx context.Context) error {
	return m.dns.Create(ctx, m.doc.OpenShiftCluster)
}

func (m *manager) ensureInfraID(ctx context.Context) (err error) {
	if m.doc.OpenShiftCluster.Properties.InfraID != "" {
		return err
	}
	// generate an infra ID that is 27 characters long with 5 bytes of them random
	infraID := generateInfraID(strings.ToLower(m.doc.OpenShiftCluster.Name), 27, 5)
	m.doc, err = m.db.PatchWithLease(ctx, m.doc.Key, func(doc *api.OpenShiftClusterDocument) error {
		doc.OpenShiftCluster.Properties.InfraID = infraID
		return nil
	})
	return err
}

func (m *manager) ensureResourceGroup(ctx context.Context) (err error) {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	group := mgmtfeatures.ResourceGroup{}

	// The FPSP's role definition does not have read on a resource group
	// if the resource group does not exist.
	// Retain the existing resource group configuration (such as tags) if it exists
	if m.doc.OpenShiftCluster.Properties.ProvisioningState != api.ProvisioningStateCreating {
		group, err = m.resourceGroups.Get(ctx, resourceGroup)
		if err != nil {
			if detailedErr, ok := err.(autorest.DetailedError); !ok || detailedErr.StatusCode != http.StatusNotFound {
				return err
			}
		}
	}

	group.Location = &m.doc.OpenShiftCluster.Location
	group.ManagedBy = &m.doc.OpenShiftCluster.ID

	// HACK: set purge=true on dev clusters so our purger wipes them out since there is not deny assignment in place
	if m.env.IsLocalDevelopmentMode() {
		if group.Tags == nil {
			group.Tags = map[string]*string{}
		}
		group.Tags["purge"] = to.StringPtr("true")
	}

	// According to https://stackoverflow.microsoft.com/a/245391/62320,
	// re-PUTting our RG should re-create RP RBAC after a customer subscription
	// migrates between tenants.
	_, err = m.resourceGroups.CreateOrUpdate(ctx, resourceGroup, group)

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

	if serviceError != nil && serviceError.Code == "ResourceGroupManagedByMismatch" {
		return &api.CloudError{
			StatusCode: http.StatusBadRequest,
			CloudErrorBody: &api.CloudErrorBody{
				Code: api.CloudErrorCodeClusterResourceGroupAlreadyExists,
				Message: "Resource group " + m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID +
					" must not already exist.",
			},
		}
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

func (m *manager) deployBaseResourceTemplate(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	infraID := m.doc.OpenShiftCluster.Properties.InfraID

	clusterStorageAccountName := "cluster" + m.doc.OpenShiftCluster.Properties.StorageSuffix
	azureRegion := strings.ToLower(m.doc.OpenShiftCluster.Location) // Used in k8s object names, so must pass DNS-1123 validation

	resources := []*arm.Resource{
		m.storageAccount(clusterStorageAccountName, azureRegion, true),
		m.storageAccountBlobContainer(clusterStorageAccountName, "ignition"),
		m.storageAccountBlobContainer(clusterStorageAccountName, "aro"),
		m.storageAccount(m.doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName, azureRegion, true),
		m.storageAccountBlobContainer(m.doc.OpenShiftCluster.Properties.ImageRegistryStorageAccountName, "image-registry"),
		m.clusterNSG(infraID, azureRegion),
		m.clusterServicePrincipalRBAC(),
		m.networkPrivateLinkService(azureRegion),
		m.networkInternalLoadBalancer(azureRegion),
	}

	// Create a public load balancer routing if needed
	if m.doc.OpenShiftCluster.Properties.NetworkProfile.OutboundType == api.OutboundTypeLoadbalancer {
		// Normal private clusters still need a public load balancer
		resources = append(resources,
			m.networkPublicIPAddress(azureRegion, infraID+"-pip-v4"),
			m.networkPublicLoadBalancer(azureRegion),
		)
		// If the cluster is public we still want the default public IP address
		if m.doc.OpenShiftCluster.Properties.IngressProfiles[0].Visibility == api.VisibilityPublic {
			resources = append(resources,
				m.networkPublicIPAddress(azureRegion, infraID+"-default-v4"),
			)
		}
	}

	if m.doc.OpenShiftCluster.Properties.FeatureProfile.GatewayEnabled {
		resources = append(resources,
			m.networkPrivateEndpoint(),
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

	return arm.DeployTemplate(ctx, m.log, m.deployments, resourceGroup, "storage", t, nil)
}

func (m *manager) attachNSGs(ctx context.Context) error {
	if m.doc.OpenShiftCluster.Properties.NetworkProfile.PreconfiguredNSG == api.PreconfiguredNSGEnabled {
		return nil
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

	return nil
}

func (m *manager) setMasterSubnetPolicies(ctx context.Context) error {
	// TODO: there is probably an undesirable race condition here - check if etags can help.
	s, err := m.subnet.Get(ctx, m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID)
	if err != nil {
		return err
	}

	if s.SubnetPropertiesFormat == nil {
		s.SubnetPropertiesFormat = &mgmtnetwork.SubnetPropertiesFormat{}
	}

	if m.doc.OpenShiftCluster.Properties.FeatureProfile.GatewayEnabled {
		s.SubnetPropertiesFormat.PrivateEndpointNetworkPolicies = to.StringPtr("Disabled")
	}
	s.SubnetPropertiesFormat.PrivateLinkServiceNetworkPolicies = to.StringPtr("Disabled")

	return m.subnet.CreateOrUpdate(ctx, m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID, s)
}

// generateInfraID take base and returns a ID that
// - is of length maxLen
// - contains randomLen random bytes
// - only contains `alphanum` or `-`
// see openshift/installer/pkg/asset/installconfig/clusterid.go for original implementation
func generateInfraID(base string, maxLen int, randomLen int) string {
	maxBaseLen := maxLen - (randomLen + 1)

	// replace all characters that are not `alphanum` or `-` with `-`
	re := regexp.MustCompile("[^A-Za-z0-9-]")
	base = re.ReplaceAllString(base, "-")

	// replace all multiple dashes in a sequence with single one.
	re = regexp.MustCompile(`-{2,}`)
	base = re.ReplaceAllString(base, "-")

	// truncate to maxBaseLen
	if len(base) > maxBaseLen {
		base = base[:maxBaseLen]
	}
	base = strings.TrimRight(base, "-")

	// add random chars to the end to randomize
	return fmt.Sprintf("%s-%s", base, utilrand.String(randomLen))
}
