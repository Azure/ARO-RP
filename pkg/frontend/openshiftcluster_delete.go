package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database/cosmosdb"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) deleteOpenShiftCluster(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)

	var header http.Header
	_, err := f.db.OpenShiftClusters.Patch(ctx, r.URL.Path, func(doc *api.OpenShiftClusterDocument) error {
		return f._deleteOpenShiftCluster(ctx, r, &header, doc)
	})
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		err = statusCodeError(http.StatusNoContent)
	case err == nil:
		err = statusCodeError(http.StatusAccepted)
	}

	reply(log, w, header, nil, err)
}

func (f *frontend) _deleteOpenShiftCluster(ctx context.Context, r *http.Request, header *http.Header, doc *api.OpenShiftClusterDocument) error {
	_, err := f.validateSubscriptionState(ctx, doc.Key, api.SubscriptionStateRegistered, api.SubscriptionStateWarned, api.SubscriptionStateSuspended)
	if err != nil {
		return err
	}

	err = validateTerminalProvisioningState(doc.OpenShiftCluster.Properties.ProvisioningState)
	if err != nil {
		return err
	}

	doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateDeleting
	doc.Dequeues = 0

	_, err = f.db.Billing.Patch(ctx, r.URL.Path, func(billingdoc *api.BillingDocument) error {
		now := time.Now()
		billingdoc.Billing.DeletionTime = &now
		return nil
	})

	if err != nil {
		return err
	}

	doc.AsyncOperationID, err = f.newAsyncOperation(ctx, r, doc)
	if err != nil {
		return err
	}

	u, err := url.Parse(r.Header.Get("Referer"))
	if err != nil {
		return err
	}

	*header = http.Header{}

	u.Path = f.operationResultsPath(r, doc.AsyncOperationID)
	(*header)["Location"] = []string{u.String()}

	u.Path = f.operationsPath(r, doc.AsyncOperationID)
	(*header)["Azure-AsyncOperation"] = []string{u.String()}

	return nil
}
