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

	mgmtfeatures "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-07-01/features"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/acrtoken"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureerrors"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	"github.com/Azure/ARO-RP/pkg/util/rbac"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

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
	"microsoft.compute/virtualmachines":     -1, // first, and before microsoft.compute/disks, microsoft.network/networkinterfaces
	"microsoft.network/privatelinkservices": -1, // before microsoft.network/loadbalancers
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
			}

			m.log.Printf("deleting %s", *resource.ID)
			future, err := m.resources.DeleteByID(ctx, *resource.ID, apiVersion)
			if err != nil {
				return err
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

func (m *manager) Delete(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')
	//In edge case of CRG not being managedBy ARO, we have a different delete path
	//we will assume normal case and set rgManagedByARO to true CRG is managedby ARO
	rgManagedByARO := true
	rg, err := m.resourceGroups.Get(ctx, resourceGroup)
	if err != nil {
		m.log.Warnf("failed to get resourceGroup %s", err)
	} else {
		if rg.ManagedBy == nil || *rg.ManagedBy == "" || !strings.EqualFold(*rg.ManagedBy, m.doc.OpenShiftCluster.ID) {
			rgManagedByARO = false
			m.log.Infof("cluster resource group not managed by aro %s", *rg.Name)
		}
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

	// only delete if managedByARO
	if rgManagedByARO {
		m.log.Printf("deleting resources")
		err = m.deleteResources(ctx)
		if err != nil {
			return err
		}

		m.log.Printf("deleting resource group %s", resourceGroup)
		err = m.resourceGroups.DeleteAndWait(ctx, resourceGroup)
		if detailedErr, ok := err.(autorest.DetailedError); ok &&
			(detailedErr.StatusCode == http.StatusForbidden || detailedErr.StatusCode == http.StatusNotFound) {
			err = nil
		}
		if azureerrors.HasAuthorizationFailedError(err) {
			err = nil
		}
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

	return m.billing.Delete(ctx, m.doc)
}
