package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/api/admin"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

const maintenanceManifestListDefaultLookback = 30 * 24 * time.Hour

func (f *frontend) getAdminMaintManifests(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	resourceID := resourceIdFromURLParams(r)
	b, err := f._getAdminMaintManifests(ctx, r, resourceID)

	if cloudErr, ok := err.(*api.CloudError); ok {
		api.WriteCloudError(w, cloudErr)
		return
	}

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getAdminMaintManifests(ctx context.Context, r *http.Request, resourceID string) ([]byte, error) {
	limitstr := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(limitstr)
	if err != nil {
		limit = 100
	}

	converter := f.apis[admin.APIVersion].MaintenanceManifestConverter

	dbOpenShiftClusters, err := f.dbGroup.OpenShiftClusters()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	dbMaintenanceManifests, err := f.dbGroup.MaintenanceManifests()
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", err.Error())
	}

	doc, err := dbOpenShiftClusters.Get(ctx, resourceID)
	if err != nil {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "", fmt.Sprintf("cluster not found: %s", err.Error()))
	}

	if doc.OpenShiftCluster.Properties.ProvisioningState == api.ProvisioningStateDeleting {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeNotFound, "", "cluster being deleted")
	}

	opts, err := parseMaintenanceManifestListOptions(r, f.now())
	if err != nil {
		return nil, err
	}

	skipToken, err := f.parseSkipToken(r.URL.String())
	if err != nil {
		return nil, err
	}

	i, err := dbMaintenanceManifests.GetByClusterResourceID(ctx, resourceID, skipToken, opts)
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

	// Embed the resolved time window in nextLink so page 2+ does not recompute
	// a drifting default after (or fail when only before was originally supplied).
	referer, err := refererWithMaintenanceManifestListOptions(r.Header.Get("Referer"), opts)
	if err != nil {
		return nil, err
	}

	nextLink, err := f.buildNextLink(referer, i.Continuation())
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(converter.ToExternalList(docList, nextLink, true), "", "    ")
}

func refererWithMaintenanceManifestListOptions(referer string, opts database.MaintenanceManifestListOptions) (string, error) {
	if referer == "" {
		return referer, nil
	}

	u, err := url.Parse(referer)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("after", strconv.FormatInt(opts.After, 10))
	if opts.Before != nil {
		q.Set("before", strconv.FormatInt(*opts.Before, 10))
	} else {
		q.Del("before")
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func parseMaintenanceManifestListOptions(r *http.Request, now time.Time) (database.MaintenanceManifestListOptions, error) {
	q := r.URL.Query()
	afterStr := q.Get("after")
	beforeStr := q.Get("before")

	var opts database.MaintenanceManifestListOptions

	if afterStr == "" && beforeStr == "" {
		opts.After = now.Add(-maintenanceManifestListDefaultLookback).Unix()
		return opts, nil
	}

	if afterStr != "" {
		after, err := strconv.ParseInt(afterStr, 10, 64)
		if err != nil {
			return opts, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "after", "after must be a unix epoch timestamp")
		}
		opts.After = after
	}

	if beforeStr != "" {
		before, err := strconv.ParseInt(beforeStr, 10, 64)
		if err != nil {
			return opts, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "before", "before must be a unix epoch timestamp")
		}
		if before < opts.After {
			return opts, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "before", "before must be greater than or equal to after")
		}
		opts.Before = &before
	}

	return opts, nil
}
