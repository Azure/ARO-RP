package backend

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/util/subnet"
)

func (b *backend) delete(ctx context.Context, log *logrus.Entry, oc *api.OpenShiftCluster) error {
	r, err := azure.ParseResourceID(oc.ID)
	if err != nil {
		return err
	}

	recordsets := dns.NewRecordSetsClient(r.SubscriptionID)
	recordsets.Authorizer = b.authorizer

	groups := resources.NewGroupsClient(r.SubscriptionID)
	groups.Authorizer = b.authorizer
	groups.Client.PollingDuration = time.Hour

	log.Printf("deleting dns")
	_, err = recordsets.Delete(ctx, os.Getenv("RESOURCEGROUP"), b.domain, "api."+oc.Properties.DomainName, dns.CNAME, "")
	if err != nil {
		return err
	}

	// TODO: ideally we would do this after all the VMs have been deleted
	for _, subnetID := range []string{
		oc.Properties.MasterProfile.SubnetID,
		oc.Properties.WorkerProfiles[0].SubnetID,
	} {
		// TODO: there is probably an undesirable race condition here - check if etags can help.
		s, err := subnet.Get(ctx, &oc.Properties.ServicePrincipalProfile, subnetID)
		if err != nil {
			return err
		}

		if s.SubnetPropertiesFormat != nil {
			s.SubnetPropertiesFormat.NetworkSecurityGroup = nil

			log.Printf("removing network security group from subnet %s", subnetID)
			err = subnet.CreateOrUpdate(ctx, &oc.Properties.ServicePrincipalProfile, subnetID, s)
			if err != nil {
				return err
			}
		}
	}

	resp, err := groups.CheckExistence(ctx, oc.Properties.ResourceGroup)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return nil
	}

	log.Printf("deleting resource group %s", oc.Properties.ResourceGroup)
	future, err := groups.Delete(ctx, oc.Properties.ResourceGroup)
	if err != nil {
		return err
	}

	log.Print("waiting for resource group deletion")
	return future.WaitForCompletionRef(ctx, groups.Client)
}
