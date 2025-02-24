package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/msi-dataplane/pkg/dataplane"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/acrtoken"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcertificates"
	azuresdkerrors "github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/errors"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	utilnet "github.com/Azure/ARO-RP/pkg/util/net"
	"github.com/Azure/ARO-RP/pkg/util/oidcbuilder"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// deleteNic deletes the network interface resource by first fetching the resource using the interface
// client, checking the provisioning state to ensure it is 'succeeded', and then deletes it
// If the nic is in a failed provisioning state, it will perform an empty CreateOrUpdate on it to put it back into
// a succeeded provisioning state.
//
// The resources client incorrectly reports provisioningState hence we must use the interface client to fetch
// this resource again so we get the correct provisioningState instead of always just "Succeeded"
func (m *manager) deleteNic(ctx context.Context, nicName string) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	nic, err := m.interfaces.Get(ctx, resourceGroup, nicName, "")

	// nic is already gone which typically happens on PLS / PE nics
	// as they are deleted in a different step
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return err
	}

	if nic.ProvisioningState == mgmtnetwork.Failed {
		m.log.Printf("NIC '%s' is in a Failed provisioning state, attempting to reconcile prior to deletion.", *nic.ID)
		err := m.interfaces.CreateOrUpdateAndWait(ctx, resourceGroup, *nic.Name, nic)
		if err != nil {
			return err
		}
	}
	return m.interfaces.DeleteAndWait(ctx, resourceGroup, *nic.Name)
}

func (m *manager) disconnectSecurityGroup(ctx context.Context, resourceID string) error {
	r, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return err
	}

	resp, err := m.armSecurityGroups.Get(ctx, r.ResourceGroup, r.ResourceName, nil)
	if err != nil {
		return err
	}
	nsg := resp.SecurityGroup

	if nsg.Properties == nil || nsg.Properties.Subnets == nil {
		return nil
	}

	for _, subnet := range nsg.Properties.Subnets {
		// Note: subnet only has value in the ID field,
		// so we have to make another API request to get full subnet struct
		// TODO: there is probably an undesirable race condition here - check if etags can help.
		s, err := m.subnet.Get(ctx, *subnet.ID)
		if err != nil {
			b, _ := json.Marshal(err)

			return &api.CloudError{
				StatusCode: http.StatusBadRequest,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidLinkedVNet,
					Message: fmt.Sprintf("Failed to get subnet '%s'.", *subnet.ID),
					Details: []api.CloudErrorBody{
						{
							Message: string(b),
						},
					},
				},
			}
		}

		if s.SubnetPropertiesFormat == nil || s.SubnetPropertiesFormat.NetworkSecurityGroup == nil ||
			!strings.EqualFold(*s.SubnetPropertiesFormat.NetworkSecurityGroup.ID, *nsg.ID) {
			continue
		}

		s.SubnetPropertiesFormat.NetworkSecurityGroup = nil

		m.log.Printf("disconnecting network security group from subnet %s", *s.ID)
		err = m.subnet.CreateOrUpdate(ctx, *s.ID, s)
		if err != nil {
			b, _ := json.Marshal(err)

			return &api.CloudError{
				StatusCode: http.StatusBadRequest,
				CloudErrorBody: &api.CloudErrorBody{
					Code:    api.CloudErrorCodeInvalidLinkedVNet,
					Message: fmt.Sprintf("Failed to update subnet '%s'.", *subnet.ID),
					Details: []api.CloudErrorBody{
						{
							Message: string(b),
						},
					},
				},
			}
		}
	}

	return nil
}

// deleteOrder maps resource types to the deletion level.  We walk the levels
// from lowest to highest, deleting all the resources in the given level in
// parallel and waiting for completion before we proceed.  Any type not in the
// map is considered to be at level 0.  Keys must be lower case.
var deleteOrder = map[string]int{
	"microsoft.compute/virtualmachines":                 -3, // first, and before microsoft.compute/disks, microsoft.network/networkinterfaces
	"microsoft.network/privatelinkservices":             -3, // before microsoft.network/loadbalancers
	"microsoft.network/privateendpoints":                -3, // before microsoft.network/networkinterfaces
	"microsoft.compute/galleries/applications/versions": -2, // before microsoft.compute/galleries/applications
	"microsoft.compute/galleries/images/versions":       -2, // before microsoft.compute/galleries/images
	"microsoft.compute/galleries/applications":          -1, // before microsoft.compute/galleries
	"microsoft.compute/galleries/images":                -1, // before microsoft.compute/galleries
	"microsoft.compute/galleries/serviceArtifacts":      -1, // before microsoft.compute/galleries
	"microsoft.network/networkinterfaces":               -1, // before microsoft.network/loadbalancers
	"microsoft.network/privatednszones":                 1,  // after everything else: get other deletions underway first
	"microsoft.compute/galleries":                       1,  // after everything else in case there are nested microsoft.compute/galleries resources
}

func (m *manager) deleteResources(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	resources, err := m.resources.ListByResourceGroup(ctx, resourceGroup, "", "", nil)
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		(detailedErr.StatusCode == http.StatusNotFound ||
			detailedErr.StatusCode == http.StatusForbidden) {
		return nil
	}
	if err != nil {
		return err
	}

	// group our resources by level
	resourceMap := map[int][]*mgmtfeatures.GenericResourceExpanded{}
	for i, resource := range resources {
		level := deleteOrder[strings.ToLower(*resource.Type)]
		resourceMap[level] = append(resourceMap[level], &resources[i])
	}

	levels := make([]int, 0, len(resourceMap))
	for level := range resourceMap {
		levels = append(levels, level)
	}
	sort.Ints(levels)

	for _, level := range levels {
		// ensure that resource deletion order is deterministic
		sort.Slice(resourceMap[level], func(i, j int) bool {
			return strings.Compare(
				strings.ToLower(*resourceMap[level][i].ID),
				strings.ToLower(*resourceMap[level][j].ID)) < 0
		})

		// asynchronously delete all resources in the level
		futures := make([]mgmtfeatures.ResourcesDeleteByIDFuture, 0, len(resourceMap[level]))
		for _, resource := range resourceMap[level] {
			apiVersion := azureclient.APIVersion(*resource.Type)
			if apiVersion == "" {
				m.log.Warnf("skipping resource %s", *resource.ID)
				continue
			}

			switch strings.ToLower(*resource.Type) {
			case "microsoft.network/networksecuritygroups":
				m.log.Printf("disconnecting network security group %s", *resource.ID)
				err = m.disconnectSecurityGroup(ctx, *resource.ID)
				if err != nil {
					return err
				}

			case "microsoft.network/privatednszones":
				m.log.Printf("deleting private DNS nested resources of %s", *resource.ID)
				err = utilnet.DeletePrivateDNSVNetLinks(ctx, m.virtualNetworkLinks, *resource.ID)
				if err != nil {
					return err
				}

			case "microsoft.network/networkinterfaces":
				err = m.deleteNic(ctx, *resource.Name)
				if err != nil {
					return err
				}
			}

			m.log.Printf("deleting %s", *resource.ID)
			future, err := m.resources.DeleteByID(ctx, *resource.ID, apiVersion)
			if err != nil {
				return deleteByIdCloudError(err)
			}

			futures = append(futures, future)
		}

		// wait for all the deletions to complete
		for i, future := range futures {
			m.log.Printf("waiting for deletion of %s", *resourceMap[level][i].ID)

			err = future.WaitForCompletionRef(ctx, m.resources.Client())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func deleteByIdCloudError(err error) error {
	detailedError, ok := err.(autorest.DetailedError)
	if !ok {
		return err
	}
	switch {
	case strings.Contains(detailedError.Original.Error(), "CannotDeleteLoadBalancerWithPrivateLinkService"):
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeCannotDeleteLoadBalancerByID,
			"features.ResourcesClient#DeleteByID", detailedError.Original.Error())

	case strings.Contains(detailedError.Original.Error(), "AuthorizationFailed"):
		return api.NewCloudError(http.StatusForbidden, api.CloudErrorCodeForbidden,
			"features.ResourcesClient#DeleteByID", detailedError.Original.Error())

	case strings.Contains(detailedError.Original.Error(), "InUseSubnetCannotBeDeleted"):
		return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInUseSubnetCannotBeDeleted,
			"features.ResourcesClient#DeleteByID", detailedError.Original.Error())

	case strings.Contains(detailedError.Original.Error(), "ScopeLocked"):
		return api.NewCloudError(http.StatusConflict, api.CloudErrorCodeScopeLocked,
			"features.ResourcesClient#DeleteByID", detailedError.Original.Error())
	}

	return err
}

func (m *manager) deleteRoleAssignments(ctx context.Context) error {
	resourceGroupID := m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	roleAssignments, err := m.roleAssignments.ListForResourceGroup(ctx, resourceGroup, "")
	if err != nil {
		return err
	}

	for _, assignment := range roleAssignments {
		if !strings.EqualFold(*assignment.Scope, resourceGroupID) ||
			strings.HasSuffix(strings.ToLower(*assignment.RoleDefinitionID), strings.ToLower(rbac.RoleOwner)) /* should only matter in development */ {
			continue
		}

		m.log.Infof("deleting role assignment %s", *assignment.Name)
		_, err := m.roleAssignments.Delete(ctx, *assignment.Scope, *assignment.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) deleteRoleDefinition(ctx context.Context) error {
	resourceGroupID := m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID

	roleDefinitions, err := m.roleDefinitions.List(ctx, resourceGroupID, "")
	if err != nil {
		return err
	}

	for _, definition := range roleDefinitions {
		if len(*definition.AssignableScopes) != 1 ||
			!strings.EqualFold((*definition.AssignableScopes)[0], resourceGroupID) ||
			!strings.HasPrefix(*definition.RoleName, "Azure Red Hat OpenShift cluster") {
			continue
		}

		m.log.Infof("deleting role definition %s", *definition.Name)
		_, err := m.roleDefinitions.Delete(ctx, (*definition.AssignableScopes)[0], *definition.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) deleteGatewayAndWait(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	if m.doc.OpenShiftCluster.Properties.NetworkProfile.GatewayPrivateLinkID == "" {
		return nil
	}

	err := m.deleteGateway(ctx)
	if err != nil {
		return err
	}

	m.log.Info("waiting for gateway record deletion")
	return wait.PollUntilContextCancel(timeoutCtx, 15*time.Second, true, func(ctx context.Context) (bool, error) {
		_, err := m.dbGateway.Get(ctx, m.doc.OpenShiftCluster.Properties.NetworkProfile.GatewayPrivateLinkID)
		if err != nil && cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
			return true, err
		}
		return false, nil
	})
}

func (m *manager) deleteGateway(ctx context.Context) error {
	if m.doc.OpenShiftCluster.Properties.NetworkProfile.GatewayPrivateLinkID == "" {
		return nil
	}

	// https://docs.microsoft.com/en-us/azure/cosmos-db/change-feed-design-patterns#deletes
	_, err := m.dbGateway.Patch(ctx, m.doc.OpenShiftCluster.Properties.NetworkProfile.GatewayPrivateLinkID, func(doc *api.GatewayDocument) error {
		doc.Gateway.Deleting = true
		doc.TTL = 60
		return nil
	})
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) /* already gone */ {
		return err
	}

	return nil
}

func (m *manager) deleteClusterMsiCertificate(ctx context.Context) error {
	// The cluster MSI may have been deleted prior to cluster deletion. If that's the case
	// we will have already deleted the certificate.
	if !m.doc.OpenShiftCluster.HasUserAssignedIdentities() {
		m.log.Warning("skipping cluster MSI certificate deletion because cluster MSI has already been deleted")
		return nil
	}

	secretName := dataplane.IdentifierForManagedIdentityCredentials(m.doc.ID)

	if _, err := m.clusterMsiKeyVaultStore.DeleteSecret(ctx, secretName, nil); err != nil && !azureerrors.IsNotFoundError(err) {
		return err
	}

	return nil
}

func (m *manager) deleteFederatedCredentials(ctx context.Context) error {
	if !m.doc.OpenShiftCluster.UsesWorkloadIdentity() || m.doc.OpenShiftCluster.Properties.ClusterProfile.OIDCIssuer == nil {
		return nil
	}

	if m.clusterMsiFederatedIdentityCredentials == nil {
		m.log.Warning("cluster MSI federated identity credentials client is nil, trying to initialize")
		err := m.initializeClusterMsiClients(ctx)
		if err != nil {
			m.log.Errorf("cluster MSI federated identity credentials client initialization failed with error: %v", err)
			return nil
		}
	}

	for _, identity := range m.doc.OpenShiftCluster.Properties.PlatformWorkloadIdentityProfile.PlatformWorkloadIdentities {
		identityResourceId, err := azure.ParseResourceID(identity.ResourceID)
		if err != nil {
			return err
		}

		federatedCredentials, err := m.clusterMsiFederatedIdentityCredentials.List(
			ctx,
			identityResourceId.ResourceGroup,
			identityResourceId.ResourceName,
			&armmsi.FederatedIdentityCredentialsClientListOptions{},
		)
		if err != nil {
			if azuresdkerrors.IsNotFoundError(err) {
				m.log.Infof("federated identity credentials not found for %s: %v", identity.ResourceID, err.Error())
			} else {
				m.log.Errorf("failed to list federated identity credentials for %s: %v", identity.ResourceID, err.Error())
			}
			continue
		}

		for _, federatedCredential := range federatedCredentials {
			switch {
			case federatedCredential == nil,
				federatedCredential.Name == nil,
				federatedCredential.Properties == nil,
				len(federatedCredential.Properties.Audiences) != 1,
				*federatedCredential.Properties.Audiences[0] != "openshift",
				federatedCredential.Properties.Issuer == nil,
				*federatedCredential.Properties.Issuer != string(*m.doc.OpenShiftCluster.Properties.ClusterProfile.OIDCIssuer):
				continue
			default:
				_, err = m.clusterMsiFederatedIdentityCredentials.Delete(
					ctx,
					identityResourceId.ResourceGroup,
					identityResourceId.ResourceName,
					*federatedCredential.Name,
					&armmsi.FederatedIdentityCredentialsClientDeleteOptions{},
				)
				if err != nil {
					if azuresdkerrors.IsNotFoundError(err) {
						m.log.Infof("federated identity credentials not found for %s: %v", identity.ResourceID, err.Error())
					} else {
						m.log.Errorf("failed to delete federated identity credentials for %s: %v", identity.ResourceID, err.Error())
					}
				}
			}
		}
	}

	return nil
}

func (m *manager) deleteResourcesAndResourceGroup(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	shouldDelete, err := m.shouldDeleteResourceGroup(ctx, resourceGroup)
	if err != nil || !shouldDelete {
		return err
	}

	m.log.Printf("deleting resources")
	err = m.deleteResources(ctx)
	if err != nil {
		return err
	}

	m.log.Printf("deleting resource group %s", resourceGroup)
	return m.deleteResourceGroup(ctx, resourceGroup)
}

func (m *manager) shouldDeleteResourceGroup(ctx context.Context, name string) (bool, error) {
	resourceGroup, err := m.resourceGroups.Get(ctx, name)
	if err != nil {
		detailedErr, isDetailedErr := err.(autorest.DetailedError)
		if azureerrors.ResourceGroupNotFound(err) || (isDetailedErr && detailedErr.StatusCode == http.StatusNotFound) {
			m.log.Infof("managed resource group %s not found, skipping deletion", name)
			err = nil
		}
		return false, err
	}

	rgManagedByCluster := resourceGroup.ManagedBy != nil && strings.EqualFold(*resourceGroup.ManagedBy, m.doc.OpenShiftCluster.ID)
	if !rgManagedByCluster {
		m.log.Infof("managed resource group %s not managed by cluster, skipping deletion", *resourceGroup.Name)
		return false, nil
	}

	return true, nil
}

func (m *manager) deleteResourceGroup(ctx context.Context, name string) error {
	err := m.resourceGroups.DeleteAndWait(ctx, name)

	detailedErr, isDetailedErr := err.(autorest.DetailedError)
	if azureerrors.ResourceGroupNotFound(err) || (isDetailedErr && (detailedErr.StatusCode == http.StatusNotFound)) {
		return nil
	}
	return err
}

func (m *manager) Delete(ctx context.Context) error {
	m.log.Printf("running ensureResourceGroup")
	err := m.ensureResourceGroup(ctx) // re-create RP RBAC if needed/missing on best-effort basics
	if err != nil {
		m.log.Error(err)
	}

	m.log.Printf("deleting dns")
	err = m.dns.Delete(ctx, m.doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	m.log.Print("deleting private endpoint")
	err = m.fpPrivateEndpoints.DeleteAndWait(ctx, m.env.ResourceGroup(), env.RPPrivateEndpointPrefix+m.doc.ID)
	if err != nil {
		return err
	}

	if m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		m.log.Printf("deleting OIDC configuration")
		blobContainerURL := oidcbuilder.GenerateBlobContainerURL(m.env)
		blobsClient, err := m.rpBlob.GetBlobsClient(blobContainerURL)
		if err != nil {
			return err
		}
		err = oidcbuilder.DeleteOidcFolder(ctx, oidcbuilder.GetBlobName(m.subscriptionDoc.Subscription.Properties.TenantID, m.doc.ID), blobsClient)
		if err != nil {
			return err
		}
	}

	m.log.Printf("deleting role assignments")
	err = m.deleteRoleAssignments(ctx)
	if err != nil {
		return err
	}

	m.log.Printf("deleting role definition")
	err = m.deleteRoleDefinition(ctx)
	if err != nil {
		return err
	}

	// private endpoint LinkIDs are reused so we wait for the deletion of the
	// gateway LinkID record before deleting the private endpoint
	// this ensures that we don't delete a LinkID record that was previously in use
	// on a newly created cluster
	m.log.Printf("deleting gateway record")
	err = m.deleteGatewayAndWait(ctx)
	if err != nil {
		return err
	}

	err = m.deleteResourcesAndResourceGroup(ctx)
	if err != nil {
		return err
	}

	if m.doc.OpenShiftCluster.UsesWorkloadIdentity() {
		m.log.Printf("deleting platform managed identities' federated credentials")
		err = m.deleteFederatedCredentials(ctx)
		if err != nil {
			return err
		}

		m.log.Printf("deleting cluster MSI certificate")
		err = m.deleteClusterMsiCertificate(ctx)
		if err != nil {
			return err
		}
	}

	if !m.env.FeatureIsSet(env.FeatureDisableSignedCertificates) {
		managedDomain, err := dns.ManagedDomain(m.env, m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain)
		if err != nil {
			return err
		}

		if managedDomain != "" {
			m.log.Print("deleting signed apiserver certificate")
			_, err = m.env.ClusterCertificates().DeleteCertificate(ctx, m.APICertName(), nil)
			if err != nil && !azcertificates.IsCertificateNotFoundError(err) {
				return err
			}

			m.log.Print("deleting signed ingress certificate")
			_, err = m.env.ClusterCertificates().DeleteCertificate(ctx, m.IngressCertName(), nil)
			if err != nil && !azcertificates.IsCertificateNotFoundError(err) {
				return err
			}
		}
	}

	if !m.env.IsLocalDevelopmentMode() {
		acrManager, err := acrtoken.NewManager(m.env, m.localFpAuthorizer)
		if err != nil {
			return err
		}

		rp := acrManager.GetRegistryProfile(m.doc.OpenShiftCluster)
		if rp != nil {
			err = acrManager.Delete(ctx, rp)
			if err != nil {
				return err
			}
		}
	}

	if m.adoptViaHive || m.installViaHive {
		err = m.hiveDeleteResources(ctx)
		if err != nil {
			return err
		}
	}

	return m.billing.Delete(ctx, m.doc)
}
