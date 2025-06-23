package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) getAdminHiveSyncSet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	syncsetname := chi.URLParam(r, "syncsetname")
	namespace := r.URL.Query().Get("cdnamespace")

	isSyncSet, err := strconv.ParseBool(r.URL.Query().Get("issyncset"))
	if err != nil {
		cloudErr := api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "invalid paremeter value: issyncset")
		api.WriteCloudError(w, cloudErr)
		return
	}

	b, err := f._getAdminHiveSyncSet(ctx, namespace, syncsetname, isSyncSet)

	if cloudErr, ok := err.(*api.CloudError); ok {
		api.WriteCloudError(w, cloudErr)
		return
	}

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _getAdminHiveSyncSet(ctx context.Context, namespace string, syncsetname string, isSyncSet bool) ([]byte, error) {
	// we have to check if the frontend has a valid syncSetManager since hive is not everywhere.
	if f.hiveSyncSetManager == nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "hive is not enabled")
	}

	if isSyncSet && namespace == "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "namespace cannot be null for getting a syncset")
	}
	if !isSyncSet && namespace != "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "namespace should be null for getting a selectorsyncset")
	}
	if syncsetname == "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "syncsetname cannot be null")
	}

	var ss interface{}
	var err error
	if isSyncSet {
		ss, err = f.hiveSyncSetManager.GetSyncSet(ctx, namespace, syncsetname)
	} else {
		ss, err = f.hiveSyncSetManager.GetSelectorSyncSet(ctx, namespace, syncsetname)
	}
	if err != nil {
		return nil, err
	}

	var b []byte
	err = codec.NewEncoderBytes(&b, &codec.JsonHandle{}).Encode(ss)
	if err != nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "unable to marshal response")
	}

	return b, nil
}
