package backend

import (
	"context"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
)

func (b *backend) delete(ctx context.Context, log *logrus.Entry, doc *api.OpenShiftClusterDocument) error {
	recordsets := dns.NewRecordSetsClient(doc.SubscriptionID)
	recordsets.Authorizer = b.authorizer

	groups := resources.NewGroupsClient(doc.SubscriptionID)
	groups.Authorizer = b.authorizer

	log.Printf("deleting dns")
	_, err := recordsets.Delete(ctx, os.Getenv("DOMAIN_RESOURCEGROUP"), os.Getenv("DOMAIN"), "api."+doc.OpenShiftCluster.Name, dns.CNAME, "")
	if err != nil {
		return err
	}

	resp, err := groups.CheckExistence(ctx, doc.OpenShiftCluster.Properties.ResourceGroup)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return nil
	}

	log.Printf("deleting resource group %s", doc.OpenShiftCluster.Properties.ResourceGroup)
	future, err := groups.Delete(ctx, doc.OpenShiftCluster.Properties.ResourceGroup)
	if err != nil {
		return err
	}

	log.Print("waiting for resource group deletion")
	return future.WaitForCompletionRef(ctx, groups.Client)
}
