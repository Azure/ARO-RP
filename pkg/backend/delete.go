package backend

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
)

func (b *backend) delete(ctx context.Context, log *logrus.Entry, doc *api.OpenShiftClusterDocument) error {
	groups := resources.NewGroupsClient(doc.SubscriptionID)
	groups.Authorizer = b.authorizer

	if doc.OpenShiftCluster.Properties.ResourceGroup == "" {
		return nil
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
