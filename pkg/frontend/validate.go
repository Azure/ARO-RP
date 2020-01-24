package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
)

func validateTerminalProvisioningState(state api.ProvisioningState) error {
	switch state {
	case api.ProvisioningStateSucceeded, api.ProvisioningStateFailed:
		return nil
	}

	return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeRequestNotAllowed, "", "Request is not allowed in provisioningState '%s'.", state)
}

func (f *frontend) validateSubscriptionState(ctx context.Context, key string, allowedStates ...api.SubscriptionState) (*api.SubscriptionDocument, error) {
	r, err := azure.ParseResourceID(key)
	if err != nil {
		return nil, err
	}

	doc, err := f.db.Subscriptions.Get(ctx, r.SubscriptionID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidSubscriptionState, "", "Request is not allowed in unregistered subscription '%s'.", r.SubscriptionID)
	case err != nil:
		return nil, err
	}

	for _, allowedState := range allowedStates {
		if doc.Subscription.State == allowedState {
			return doc, nil
		}
	}

	return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidSubscriptionState, "", "Request is not allowed in subscription in state '%s'.", doc.Subscription.State)
}
