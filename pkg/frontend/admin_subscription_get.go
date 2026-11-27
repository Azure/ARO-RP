package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getAdminSubscription(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	subscriptionID := chi.URLParam(r, "subscriptionId")

	b, err := f._getAdminSubscription(ctx, subscriptionID)

	if cloudErr, ok := err.(*api.CloudError); ok {
		api.WriteCloudError(w, cloudErr)
		return
	}

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getAdminSubscription(ctx context.Context, subscriptionID string) ([]byte, error) {
	dbSubscriptions, err := f.dbGroup.Subscriptions()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	doc, err := dbSubscriptions.Get(ctx, subscriptionID)
	if err != nil && cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "", "subscription not found")
	} else if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	return f.encodeToJSON(doc.Subscription)
}
