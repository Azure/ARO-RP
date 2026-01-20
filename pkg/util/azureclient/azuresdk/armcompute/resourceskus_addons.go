package armcompute

import (
	"context"

	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

type ResourceSKUsClientAddons interface {
	List(ctx context.Context, filter string, includeExtendedLocations bool) ([]*armcompute.ResourceSKU, error)
}

func (c *resourceSKUsClient) List(ctx context.Context, filter string, includeExtendedLocations bool) (result []*armcompute.ResourceSKU, err error) {
	ex := "false"
	if includeExtendedLocations {
		ex = "true"
	}

	pager := c.NewListPager(&armcompute.ResourceSKUsClientListOptions{
		Filter:                   pointerutils.ToPtr(filter),
		IncludeExtendedLocations: pointerutils.ToPtr(ex),
	})

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, page.Value...)
	}

	return result, nil
}
