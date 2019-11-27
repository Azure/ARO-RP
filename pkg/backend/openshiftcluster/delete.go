package openshiftcluster

import (
	"context"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"

	"github.com/jim-minter/rp/pkg/util/subnet"
)

func (m *Manager) Delete(ctx context.Context) error {
	m.log.Printf("deleting dns")
	_, err := m.recordsets.Delete(ctx, os.Getenv("RESOURCEGROUP"), m.domain, "api."+m.oc.Properties.DomainName, dns.CNAME, "")
	if err != nil {
		return err
	}

	// TODO: ideally we would do this after all the VMs have been deleted
	for _, subnetID := range []string{
		m.oc.Properties.MasterProfile.SubnetID,
		m.oc.Properties.WorkerProfiles[0].SubnetID,
	} {
		// TODO: there is probably an undesirable race condition here - check if etags can help.
		s, err := subnet.Get(ctx, &m.oc.Properties.ServicePrincipalProfile, subnetID)
		if err != nil {
			return err
		}

		if s.SubnetPropertiesFormat != nil {
			s.SubnetPropertiesFormat.NetworkSecurityGroup = nil

			m.log.Printf("removing network security group from subnet %s", subnetID)
			err = subnet.CreateOrUpdate(ctx, &m.oc.Properties.ServicePrincipalProfile, subnetID, s)
			if err != nil {
				return err
			}
		}
	}

	resp, err := m.groups.CheckExistence(ctx, m.oc.Properties.ResourceGroup)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return nil
	}

	m.log.Printf("deleting resource group %s", m.oc.Properties.ResourceGroup)
	future, err := m.groups.Delete(ctx, m.oc.Properties.ResourceGroup)
	if err != nil {
		return err
	}

	m.log.Print("waiting for resource group deletion")
	return future.WaitForCompletionRef(ctx, m.groups.Client)
}
