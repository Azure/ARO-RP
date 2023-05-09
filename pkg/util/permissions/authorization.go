package permissions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/authorization"
)

type permissionsValidatorPermissionsClient struct {
	log *logrus.Entry

	permissions authorization.PermissionsClient
}

func NewPermissionsValidatorWithPermissionsClient(log *logrus.Entry, permissions authorization.PermissionsClient) PermissionsValidator {
	return &permissionsValidatorPermissionsClient{
		log:         log,
		permissions: permissions,
	}
}

func (c *permissionsValidatorPermissionsClient) ValidateActions(ctx context.Context, r *azure.Resource, actions []string) error {
	timeout := 20 * time.Second
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return wait.PollImmediateUntil(timeout, func() (bool, error) {
		return c.usingListPermissions(ctx, r, actions)
	}, timeoutCtx.Done())
}

// usingListPermissions is how the current check is done
func (c permissionsValidatorPermissionsClient) usingListPermissions(
	ctx context.Context,
	resource *azure.Resource,
	actions []string,
) (bool, error) {
	c.log.Debug("retry validateActions with ListPermissions")
	perms, err := c.permissions.ListForResource(
		ctx,
		resource.ResourceGroup,
		resource.Provider,
		"",
		resource.ResourceType,
		resource.ResourceName,
	)
	if err != nil {
		return false, err
	}

	for _, action := range actions {
		ok, err := canDoAction(perms, action)
		if !ok || err != nil {
			// TODO(jminter): I don't understand if there are genuinely
			// cases where CanDoAction can return false then true shortly
			// after. I'm a little skeptical; if it can't happen we can
			// simplify this code.  We should add a metric on this.
			return false, err
		}
	}
	return true, nil
}
