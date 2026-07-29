package armredhatopenshift

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	armredhatopenshift "github.com/Azure/ARO-RP/pkg/client/sdk/resourcemanager/redhatopenshift/armredhatopenshift"
)

// OperationsAddons provides additional methods for OperationsClient
type OperationsAddons interface {
	List(ctx context.Context, options *armredhatopenshift.OperationsClientListOptions) ([]*armredhatopenshift.Operation, error)
}

// List iterates through all available operations
func (c *operationsClient) List(ctx context.Context, options *armredhatopenshift.OperationsClientListOptions) (result []*armredhatopenshift.Operation, err error) {
	pager := c.NewListPager(options)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, page.Value...)
	}
	return result, nil
}
