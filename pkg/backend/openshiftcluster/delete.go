package openshiftcluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func (m *Manager) Delete(ctx context.Context) error {
	resourceGroup := stringutils.LastTokenByte(m.doc.OpenShiftCluster.Properties.ClusterProfile.ResourceGroupID, '/')

	m.log.Printf("deleting dns")
	err := m.dns.Delete(ctx, m.doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	// TODO: ideally we would do this after all the VMs have been deleted
	for _, subnetID := range []string{
		m.doc.OpenShiftCluster.Properties.MasterProfile.SubnetID,
		m.doc.OpenShiftCluster.Properties.WorkerProfiles[0].SubnetID,
	} {
		// TODO: there is probably an undesirable race condition here - check if etags can help.
		s, err := m.subnet.Get(ctx, subnetID)
		if err != nil {
			m.log.Error(err)
			continue
		}

		nsgID, err := subnet.NetworkSecurityGroupID(m.doc.OpenShiftCluster, subnetID)
		if err != nil {
			m.log.Error(err)
			continue
		}

		if s.SubnetPropertiesFormat == nil ||
			s.SubnetPropertiesFormat.NetworkSecurityGroup == nil ||
			!strings.EqualFold(*s.SubnetPropertiesFormat.NetworkSecurityGroup.ID, nsgID) {
			continue
		}

		s.SubnetPropertiesFormat.NetworkSecurityGroup = nil

		m.log.Printf("removing network security group from subnet %s", subnetID)
		err = m.subnet.CreateOrUpdate(ctx, subnetID, s)
		if err != nil {
			m.log.Error(err)
			continue
		}
	}

	m.log.Print("deleting private endpoint")
	err = m.privateendpoint.Delete(ctx, m.doc)
	if err != nil {
		return err
	}

	if _, ok := m.env.(env.Dev); !ok {
		managedDomain, err := m.env.ManagedDomain(m.doc.OpenShiftCluster.Properties.ClusterProfile.Domain)
		if err != nil {
			return err
		}

		if managedDomain != "" {
			m.log.Print("deleting signed apiserver certificate")
			err = m.keyvault.DeleteCertificate(ctx, m.doc.ID+"-apiserver")
			if err != nil {
				return err
			}

			m.log.Print("deleting signed ingress certificate")
			err = m.keyvault.DeleteCertificate(ctx, m.doc.ID+"-ingress")
			if err != nil {
				return err
			}
		}
	}

	m.log.Printf("deleting resource group %s", resourceGroup)
	err = m.groups.DeleteAndWait(ctx, resourceGroup)
	if detailedErr, ok := err.(autorest.DetailedError); ok &&
		detailedErr.StatusCode == http.StatusForbidden {
		err = nil
	}

	if err != nil {
		return err
	}

	m.log.Printf("updating billing record with deletion time")
	_, err = m.billing.MarkForDeletion(ctx, m.doc.ID)

	return err
}
