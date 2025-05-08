package redhatopenshift

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mgmtredhatopenshift20231122 "github.com/Azure/ARO-RP/pkg/client/services/redhatopenshift/mgmt/2023-11-22/redhatopenshift"
)

// OperationsClientAddons contains addons for OperationsClient
type OperationsClientAddons interface {
	List(ctx context.Context) (operations []mgmtredhatopenshift20231122.Operation, err error)
}

func (c *operationsClient) List(ctx context.Context) (operations []mgmtredhatopenshift20231122.Operation, err error) {
	page, err := c.OperationsClient.List(ctx)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		operations = append(operations, page.Values()...)

		err = page.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return operations, nil
}
