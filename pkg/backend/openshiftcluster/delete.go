package openshiftcluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

func (m *Manager) Delete(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	m.log.Printf("deleting dns")
	err := m.dns.Delete(ctx, m.doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	m.log.Print("looking for network security groups to remove from subnets")
	nsgs, err := m.securityGroups.List(ctx, resourceGroup)
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusNotFound {
		err = nil
	}
	if err != nil {
		return err
	}

	// TODO: ideally we would do this after all the VMs have been deleted
	for _, nsg := range nsgs {
		if nsg.SecurityGroupPropertiesFormat == nil ||
			nsg.SecurityGroupPropertiesFormat.Subnets == nil {
			continue
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

			m.log.Printf("removing network security group from subnet %s", *s.ID)
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
	}

	m.log.Print("deleting private endpoint")
	err = m.privateendpoint.Delete(ctx, m.doc)
	if err != nil {
		return err
	}

	if m.env.DeploymentMode() != deployment.Development {
		managedDomain, err := m.env.ManagedDomain(m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain)
		if err != nil {
			return err
		}

		if managedDomain != "" {
			m.log.Print("deleting signed apiserver certificate")
			err = m.keyvault.EnsureCertificateDeleted(ctx, m.env.ClustersKeyvaultURI(), m.doc.ID+"-apiserver")
			if err != nil {
				return err
			}

			m.log.Print("deleting signed ingress certificate")
			err = m.keyvault.EnsureCertificateDeleted(ctx, m.env.ClustersKeyvaultURI(), m.doc.ID+"-ingress")
			if err != nil {
				return err
			}
		}
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
		rp := m.acrtoken.GetRegistryProfile(m.doc.OpenShiftCluster)
		if rp != nil {
			err = m.acrtoken.Delete(ctx, rp)
			if err != nil {
				return err
			}
		}
	}

	return m.billing.Delete(ctx, m.doc)
}
