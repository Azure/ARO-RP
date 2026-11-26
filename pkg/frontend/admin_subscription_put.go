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

func (f *frontend) putAdminSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	var b []byte
	err := cosmosdb.RetryOnPreconditionFailed(func() error {
		var err error
		b, err = f._putAdminSubscription(ctx, r)
		return err
	})

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _putAdminSubscription(ctx context.Context, r *http.Request) ([]byte, error) {
	body := r.Context().Value(middleware.ContextKeyBody).([]byte)

	dbSubscriptions, err := f.dbGroup.Subscriptions()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}
	subscriptionID := chi.URLParam(r, "subscriptionId")

	doc, err := dbSubscriptions.Get(ctx, subscriptionID)
	if err != nil {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "", "subscription not found")
	}

	sub := &api.Subscription{}
	err = codec.NewDecoderBytes(body, f.indentingJsonHandle).Decode(&sub)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", fmt.Sprintf("The request content was invalid and could not be deserialized: %q.", err))
	}

	// We only apply the components of the subscription that are not controlled
	// by the ARM contract (our overrides). We deliberately do not copy the
	// rest, even if it is in the body.
	doc.Subscription.Overrides = sub.Overrides

	doc, err = dbSubscriptions.Update(ctx, doc)
	if err != nil {
		return nil, err
	}

	return f.encodeToJSON(doc.Subscription)
}
