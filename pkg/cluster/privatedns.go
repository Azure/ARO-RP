package cluster

import (
	"context"
	"errors"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/privatedns"
)

func DeletePrivateDNSVNetLinks(ctx context.Context, vNetLinksClient privatedns.VirtualNetworkLinksClient, resourceID string) error {
	r, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return err
	}

	if vNetLinksClient == nil {
		return errors.New("vNetLinksClient is nil")
	}

	vNetLinks, err := vNetLinksClient.List(ctx, r.ResourceGroup, r.ResourceName, nil)
	if err != nil {
		return err
	}

	for _, vNetLink := range vNetLinks {
		err = vNetLinksClient.DeleteAndWait(ctx, r.ResourceGroup, r.ResourceName, *vNetLink.Name, "")
		if err != nil {
			return err
		}
	}

	return nil
}
