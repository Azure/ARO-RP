package authorization

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
)

// RoleAssignmentsClientAddon is a minimal interface for azure RoleAssignmentsClientAddon
type RoleAssignmentsClientAddon interface {
	List(ctx context.Context, filter string) (result []authorization.RoleAssignment, err error)
	ListForScope(ctx context.Context, scope string, filter string) (result []authorization.RoleAssignment, err error)
}

func (ra *roleAssignmentsClient) List(ctx context.Context, filter string) (result []authorization.RoleAssignment, err error) {
	page, err := ra.RoleAssignmentsClient.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		result = append(result, page.Values()...)

		err = page.Next()
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (ra *roleAssignmentsClient) ListForScope(ctx context.Context, scope string, filter string) (result []authorization.RoleAssignment, err error) {
	page, err := ra.RoleAssignmentsClient.ListForScope(ctx, scope, filter)
	if err != nil {
		return nil, err
	}

	for page.NotDone() {
		result = append(result, page.Values()...)

		err = page.Next()
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}
