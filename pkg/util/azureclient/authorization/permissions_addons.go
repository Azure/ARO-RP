package authorization

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/Azure/go-autorest/autorest/azure"
)

// PermissionsClientAddons contains addons for PermissionsClient
type PermissionsClientAddons interface {
	ListForResource(ctx context.Context, resourceID string) (permissions []authorization.Permission, err error)
}

func (c *permissionsClient) ListForResource(ctx context.Context, resourceID string) (permissions []authorization.Permission, error error) {
	r, err := azure.ParseResourceID(resourceID)
	if err != nil {
		return nil, err
	}

	page, err := c.PermissionsClient.ListForResource(ctx, r.ResourceGroup, r.Provider, r.ResourceType, "", r.ResourceName)
	if err != nil {
		return nil, err
	}

	for {
		permissions = append(permissions, page.Values()...)

		err = page.Next()
		if err != nil {
			return nil, err
		}

		if !page.NotDone() {
			break
		}
	}

	return permissions, nil
}
