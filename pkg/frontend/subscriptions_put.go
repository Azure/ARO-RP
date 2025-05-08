package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) putSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	var b []byte
	err := cosmosdb.RetryOnPreconditionFailed(func() error {
		var err error
		b, err = f._putSubscription(ctx, r)
		return err
	})

	reply(log, w, nil, b, err)
}

func (f *frontend) _putSubscription(ctx context.Context, r *http.Request) ([]byte, error) {
	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	subId := chi.URLParam(r, "subscriptionId")

	dbSubscriptions, err := f.dbGroup.Subscriptions()
	if err != nil {
		return nil, err
	}

	doc, err := dbSubscriptions.Get(ctx, subId)
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, err
	}

	isCreate := doc == nil

	if isCreate {
		doc = &api.SubscriptionDocument{
			ID:           subId,
			Subscription: &api.Subscription{},
		}
	}

	oldState := doc.Subscription.State

	h := &codec.JsonHandle{
		Indent: 4,
	}

	doc.Subscription = &api.Subscription{}
	err = codec.NewDecoderBytes(body, h).Decode(&doc.Subscription)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", fmt.Sprintf("The request content was invalid and could not be deserialized: %q.", err))
	}

	switch doc.Subscription.State {
	case api.SubscriptionStateRegistered, api.SubscriptionStateUnregistered,
		api.SubscriptionStateWarned, api.SubscriptionStateSuspended:
		// allow
	case api.SubscriptionStateDeleted:
		doc.Deleting = true
	default:
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "state", fmt.Sprintf("The provided state '%s' is invalid.", doc.Subscription.State))
	}

	if oldState == api.SubscriptionStateDeleted && doc.Subscription.State != api.SubscriptionStateDeleted {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidSubscriptionState, "", fmt.Sprintf("Request is not allowed in subscription in state '%s'.", oldState))
	}

	if doc.Subscription.Properties != nil &&
		doc.Subscription.Properties.AccountOwner != nil &&
		doc.Subscription.Properties.AccountOwner.Email != "" {
		doc.Subscription.Properties.AccountOwner.Email = ""
	}

	if isCreate {
		doc, err = dbSubscriptions.Create(ctx, doc)
	} else {
		doc, err = dbSubscriptions.Update(ctx, doc)
	}
	if err != nil {
		return nil, err
	}

	var b []byte
	err = codec.NewEncoderBytes(&b, h).Encode(doc.Subscription)
	if err != nil {
		return nil, err
	}

	if isCreate {
		err = statusCodeError(http.StatusCreated)
	}
	return b, err
}
