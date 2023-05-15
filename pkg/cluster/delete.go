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

	mgmtnetwork "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-08-01/network"
	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/acrtoken"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	"github.com/Azure/ARO-RP/pkg/util/dns"
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

func (m *manager) deletePrivateDNSVirtualNetworkLinks(ctx context.Context, resourceID string) error {
	r, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return err
	}

	virtualNetworkLinks, err := m.virtualNetworkLinks.List(ctx, r.ResourceGroup, r.ResourceName, nil)
	if err != nil {
		return err
	}

	for _, virtualNetworkLink := range virtualNetworkLinks {
		err = m.virtualNetworkLinks.DeleteAndWait(ctx, r.ResourceGroup, r.ResourceName, *virtualNetworkLink.Name, "")
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) disconnectSecurityGroup(ctx context.Context, resourceID string) error {
	r, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return err
	}

	nsg, err := m.securityGroups.Get(ctx, r.ResourceGroup, r.ResourceName, "")
	if err != nil {
		return err
	}

	if nsg.SecurityGroupPropertiesFormat == nil ||
		nsg.SecurityGroupPropertiesFormat.Subnets == nil {
		return nil
	}

	for _, subnet := range *nsg.SecurityGroupPropertiesFormat.Subnets {
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

		if s.SubnetPropertiesFormat == nil ||
			s.SubnetPropertiesFormat.NetworkSecurityGroup == nil ||
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
	"microsoft.compute/virtualmachines":     -2, // first, and before microsoft.compute/disks, microsoft.network/networkinterfaces
	"microsoft.network/privatelinkservices": -2, // before microsoft.network/loadbalancers
	"microsoft.network/privateendpoints":    -2, // before microsoft.network/networkinterfaces
	"microsoft.network/networkinterfaces":   -1, // before microsoft.network/loadbalancers
	"microsoft.network/privatednszones":     1,  // after everything else: get other deletions underway first
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
				err = m.deletePrivateDNSVirtualNetworkLinks(ctx, *resource.ID)
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
	return wait.PollImmediateUntil(15*time.Second, func() (bool, error) {
		_, err := m.dbGateway.Get(ctx, m.doc.OpenShiftCluster.Properties.NetworkProfile.GatewayPrivateLinkID)
		if err != nil && cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) /* already gone */ {
			return true, nil
		}
		return false, nil
	}, timeoutCtx.Done())
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

func (m *manager) deleteResourcesAndResourceGroup(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	// In edge case of CRG not being managedBy ARO, we have a different delete path
	// we will assume normal case and set rgManagedByARO to true CRG is managedBy ARO
	rgManagedByARO := true
	rg, err := m.resourceGroups.Get(ctx, resourceGroup)
	if err != nil {
		m.log.Warnf("failed to get resourceGroup %s", err)
	} else if !m.env.IsLocalDevelopmentMode() {
		if rg.ManagedBy == nil || *rg.ManagedBy == "" || !strings.EqualFold(*rg.ManagedBy, m.doc.OpenShiftCluster.ID) {
			rgManagedByARO = false
			m.log.Infof("cluster resource group not managed by aro %s", *rg.Name)
		}
	}

	// Do not delete the resource group if it is not managed by ARO
	if !rgManagedByARO {
		return nil
	}

	m.log.Printf("deleting resources")
	err = m.deleteResources(ctx)
	if err != nil {
		return err
	}

	m.log.Printf("deleting resource group %s", resourceGroup)
	err = m.resourceGroups.DeleteAndWait(ctx, resourceGroup)
	detailedErr, ok := err.(autorest.DetailedError)
	if ok && (detailedErr.StatusCode == http.StatusForbidden || detailedErr.StatusCode == http.StatusNotFound) {
		err = nil
	}
	if azureerrors.HasAuthorizationFailedError(err) || azureerrors.ResourceGroupNotFound(err) {
		err = nil
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

	if !m.env.FeatureIsSet(env.FeatureDisableSignedCertificates) {
		managedDomain, err := dns.ManagedDomain(m.env, m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain)
		if err != nil {
			return err
		}

		if managedDomain != "" {
			m.log.Print("deleting signed apiserver certificate")
			err = m.env.ClusterKeyvault().EnsureCertificateDeleted(ctx, m.doc.ID+"-apiserver")
			if err != nil {
				return err
			}

			m.log.Print("deleting signed ingress certificate")
			err = m.env.ClusterKeyvault().EnsureCertificateDeleted(ctx, m.doc.ID+"-ingress")
			if err != nil {
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
		// Don't fail the deletion because of hive
		// This should change when/if we start using hive for cluster deletion
		err = m.hiveDeleteResources(ctx)
		if err != nil {
			m.log.Info(err)
		}
	}

	return m.billing.Delete(ctx, m.doc)
}
