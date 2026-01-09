package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getAdminBillingDocuments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log, ok := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	if !ok {
		log = logrus.NewEntry(logrus.StandardLogger())
	}
	r.URL.Path = filepath.Dir(r.URL.Path)

	apiVersion, ok := f.apis[admin.APIVersion]
	if !ok {
		adminReply(log, w, nil, nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "API version not found"))
		return
	}

	dbBilling, err := f.dbGroup.Billing()
	if err != nil {
		adminReply(log, w, nil, nil, err)
		return
	}

	b, err := f._getBillingDocuments(ctx, r, apiVersion.BillingDocumentConverter, func(skipToken string) (cosmosdb.BillingDocumentIterator, error) {
		return dbBilling.List(skipToken), nil
	})

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getBillingDocuments(ctx context.Context, r *http.Request, converter api.BillingDocumentConverter, lister func(string) (cosmosdb.BillingDocumentIterator, error)) ([]byte, error) {
	skipToken, err := f.parseSkipToken(r.URL.String())
	if err != nil {
		return nil, err
	}

	i, err := lister(skipToken)
	if err != nil {
		return nil, err
	}

	docs, err := i.Next(ctx, 10)
	if err != nil {
		return nil, err
	}

	var billingDocs []*api.BillingDocument
	if docs != nil {
		billingDocs = docs.BillingDocuments
	}

	nextLink, err := f.buildNextLink(r.Header.Get("Referer"), i.Continuation())
	if err != nil {
		return nil, err
	}

	return json.Marshal(converter.ToExternalList(billingDocs, nextLink))
}
