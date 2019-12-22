package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) putSubscription(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(middleware.ContextKeyLog).(*logrus.Entry)

	var b []byte
	var created bool
	err := cosmosdb.RetryOnPreconditionFailed(func() error {
		var err error
		b, created, err = f._putSubscription(r)
		return err
	})
	if err == nil && created {
		w.WriteHeader(http.StatusCreated)
	}

	reply(log, w, b, err)
}

func (f *frontend) _putSubscription(r *http.Request) ([]byte, bool, error) {
	body := r.Context().Value(middleware.ContextKeyBody).([]byte)
	vars := mux.Vars(r)

	doc, err := f.db.Subscriptions.Get(vars["subscriptionId"])
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, false, err
	}

	isCreate := doc == nil

	if isCreate {
		doc = &api.SubscriptionDocument{
			ID:           vars["subscriptionId"],
			Subscription: &api.Subscription{},
		}
	}

	oldState := doc.Subscription.State

	h := &codec.JsonHandle{
		BasicHandle: codec.BasicHandle{
			DecodeOptions: codec.DecodeOptions{
				ErrorIfNoField: true,
			},
		},
		Indent: 4,
	}

	doc.Subscription = &api.Subscription{}
	err = codec.NewDecoderBytes(body, h).Decode(&doc.Subscription)
	if err != nil {
		return nil, false, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
	}

	switch doc.Subscription.State {
	case api.SubscriptionStateRegistered, api.SubscriptionStateUnregistered,
		api.SubscriptionStateWarned, api.SubscriptionStateSuspended:
		// allow
	case api.SubscriptionStateDeleted:
		doc.Deleting = true
	default:
		return nil, false, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "state", "The provided state '%s' is invalid.", doc.Subscription.State)
	}

	if oldState == api.SubscriptionStateDeleted && doc.Subscription.State != api.SubscriptionStateDeleted {
		return nil, false, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidSubscriptionState, "", "Request is not allowed in subscription in state '%s'.", oldState)
	}

	if isCreate {
		doc, err = f.db.Subscriptions.Create(doc)
	} else {
		doc, err = f.db.Subscriptions.Update(doc)
	}
	if err != nil {
		return nil, false, err
	}

	var b []byte
	err = codec.NewEncoderBytes(&b, h).Encode(doc.Subscription)
	if err != nil {
		return nil, false, err
	}

	return b, isCreate, nil
}
