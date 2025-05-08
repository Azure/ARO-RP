package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getAdminQueuedMaintManifests(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	b, err := f._getAdminQueuedMaintManifests(ctx, r)

	if cloudErr, ok := err.(*api.CloudError); ok {
		api.WriteCloudError(w, cloudErr)
		return
	}

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getAdminQueuedMaintManifests(ctx context.Context, r *http.Request) ([]byte, error) {
	limitstr := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(limitstr)
	if err != nil {
		limit = 100
	}

	converter := f.apis[admin.APIVersion].MaintenanceManifestConverter

	dbMaintenanceManifests, err := f.dbGroup.MaintenanceManifests()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	skipToken, err := f.parseSkipToken(r.URL.String())
	if err != nil {
		return nil, err
	}

	i, err := dbMaintenanceManifests.Queued(ctx, skipToken)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	docList := make([]*api.MaintenanceManifestDocument, 0)
	for {
		docs, err := i.Next(ctx, int(math.Min(float64(limit), 10)))
		if err != nil {
			return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", fmt.Errorf("failed reading next manifest document: %w", err).Error())
		}
		if docs == nil {
			break
		}

		docList = append(docList, docs.MaintenanceManifestDocuments...)

		if len(docList) >= limit {
			break
		}
	}

	nextLink, err := f.buildNextLink(r.Header.Get("Referer"), i.Continuation())
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(converter.ToExternalList(docList, nextLink, false), "", "    ")
}
