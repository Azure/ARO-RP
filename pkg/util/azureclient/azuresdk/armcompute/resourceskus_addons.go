package armcompute

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"iter"

	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"

	"github.com/Azure/ARO-RP/pkg/util/pointerutils"
)

type ResourceSKUsClientAddons interface {
	List(ctx context.Context, filter string, includeExtendedLocations bool) iter.Seq2[*armcompute.ResourceSKU, error]
}

func (c *resourceSKUsClient) List(ctx context.Context, filter string, includeExtendedLocations bool) iter.Seq2[*armcompute.ResourceSKU, error] {
	ex := "false"
	if includeExtendedLocations {
		ex = "true"
	}

	pager := c.NewListPager(&armcompute.ResourceSKUsClientListOptions{
		Filter:                   pointerutils.ToPtr(filter),
		IncludeExtendedLocations: pointerutils.ToPtr(ex),
	})

	return func(yield func(*armcompute.ResourceSKU, error) bool) {
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				yield(nil, err)
				return
			}

			for _, v := range page.Value {
				if !yield(v, nil) {
					return
				}
			}
		}
	}
}
