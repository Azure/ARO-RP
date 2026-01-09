package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getAdminBillingDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log, ok := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	if !ok {
		log = logrus.NewEntry(logrus.StandardLogger())
	}

	b, err := f._getAdminBillingDocument(ctx, r)

	var cloudErr *api.CloudError
	if errors.As(err, &cloudErr) {
		api.WriteCloudError(w, cloudErr)
		return
	}

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getAdminBillingDocument(ctx context.Context, r *http.Request) ([]byte, error) {
	billingDocId := chi.URLParam(r, "billingDocId")

	apiVersion, ok := f.apis[admin.APIVersion]
	if !ok {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "API version not found")
	}

	dbBilling, err := f.dbGroup.Billing()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	doc, err := dbBilling.Get(ctx, billingDocId)
	if err != nil {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "", fmt.Sprintf("billing document not found: %s", err.Error()))
	}

	return json.Marshal(apiVersion.BillingDocumentConverter.ToExternal(doc))
}
