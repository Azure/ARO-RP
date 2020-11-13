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
	"github.com/Azure/ARO-RP/pkg/util/acrtoken"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/dns"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *manager) deletePrivateDNSVirtualNetworkLinks(ctx context.Context, resourceID string) error {
	r, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return err
	}

	virtualnetworklinks, err := m.virtualnetworklinks.List(ctx, r.ResourceGroup, r.ResourceName, nil)
	if err != nil {
		return err
	}

	for _, virtualnetworklink := range virtualnetworklinks {
		err = m.virtualnetworklinks.DeleteAndWait(ctx, r.ResourceGroup, r.ResourceName, *virtualnetworklink.Name, "")
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

	nsg, err := m.securitygroups.Get(ctx, r.ResourceGroup, r.ResourceName, "")
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

// keys must be lower case
var deleteOrder = map[string]int{
	"microsoft.compute/virtualmachines":     -1, // first, and before microsoft.compute/disks, microsoft.network/networkinterfaces
	"microsoft.network/privatelinkservices": -1, // before microsoft.network/loadbalancers
	"microsoft.network/privatednszones":     1,  // after everything else: get other deletions underway first
}

func (m *manager) deleteResources(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	resources, err := m.resources.ListByResourceGroup(ctx, resourceGroup, "", "", nil)
	if err != nil {
		return err
	}

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
		sort.Slice(resourceMap[level], func(i, j int) bool {
			return strings.Compare(
				strings.ToLower(*resourceMap[level][i].ID),
				strings.ToLower(*resourceMap[level][j].ID)) < 0
		})

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

func (m *manager) Delete(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	m.log.Printf("deleting dns")
	err := m.dns.Delete(ctx, m.doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	m.log.Print("deleting private endpoint")
	err = m.privateendpoint.Delete(ctx, m.doc)
	if err != nil {
		return err
	}

	m.log.Printf("deleting resources")
	err = m.deleteResources(ctx)
	if err != nil {
		return err
	}

	m.log.Printf("deleting resource group %s", resourceGroup)
	err = m.groups.DeleteAndWait(ctx, resourceGroup)
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		(detailedErr.StatusCode == http.StatusForbidden || detailedErr.StatusCode == http.StatusNotFound) {
		err = nil
	}
	if err != nil {
		return err
	}

	if m.env.DeploymentMode() != deployment.Development {
		managedDomain, err := dns.ManagedDomain(m.env, m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain)
		if err != nil {
			return err
		}

		if managedDomain != "" {
			m.log.Print("deleting signed apiserver certificate")
			err = m.keyvault.EnsureCertificateDeleted(ctx, m.doc.ID+"-apiserver")
			if err != nil {
				return err
			}

			m.log.Print("deleting signed ingress certificate")
			err = m.keyvault.EnsureCertificateDeleted(ctx, m.doc.ID+"-ingress")
			if err != nil {
				return err
			}
		}

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
