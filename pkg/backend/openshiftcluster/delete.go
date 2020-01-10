package openshiftcluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strings"

	"github.com/Azure/go-autorest/autorest"

	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

func (m *Manager) Delete(ctx context.Context) error {
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
		s, err := m.subnets.Get(ctx, subnetID)
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
		err = m.subnets.CreateOrUpdate(ctx, subnetID, s)
		if err != nil {
			m.log.Error(err)
			continue
		}
	}

	_, err = m.groups.CheckExistence(ctx, m.doc.OpenShiftCluster.Properties.ResourceGroup)
	if err != nil {
		if detailedErr, ok := err.(autorest.DetailedError); ok {
			if detailedErr.StatusCode == http.StatusForbidden {
				return nil
			}
		}
		return err
	}

	m.log.Print("deleting private endpoint")
	err = m.privateendpoint.Delete(ctx, m.doc)
	if err != nil {
		return err
	}

	m.log.Printf("deleting resource group %s", m.doc.OpenShiftCluster.Properties.ResourceGroup)
	return m.groups.DeleteAndWait(ctx, m.doc.OpenShiftCluster.Properties.ResourceGroup)
}
