package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"reflect"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	hivev1 "github.com/openshift/hive/apis/hive/v1"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
)

func (f *frontend) listAdminHiveSyncSet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := ctx.Value(middleware.ContextKeyLog).(*logrus.Entry)
	namespace := r.URL.Query().Get("cdnamespace")
	label := r.URL.Query().Get("label")

	isSyncSet, err := strconv.ParseBool(r.URL.Query().Get("issyncset"))
	if err != nil {
		cloudErr := api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "invalid paremeter value: issyncset")
		api.WriteCloudError(w, cloudErr)
		return
	}

	b, err := f._listAdminHiveSyncSet(ctx, namespace, label, isSyncSet)

	if cloudErr, ok := err.(*api.CloudError); ok {
		api.WriteCloudError(w, cloudErr)
		return
	}

	adminReply(log, w, nil, b, err)
}

func (f *frontend) _listAdminHiveSyncSet(ctx context.Context, namespace string, label string, isSyncSet bool) ([]byte, error) {
	// we have to check if the frontend has a valid syncSetManager since hive is not everywhere.
	if f.hiveSyncSetManager == nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "hive is not enabled")
	}

	// defaults to listing selectorSyncSets for an AKS instance
	ssType := reflect.TypeOf(hivev1.SelectorSyncSetList{})
	if isSyncSet {
		ssType = reflect.TypeOf(hivev1.SyncSetList{})
	}
	if isSyncSet && namespace == "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "namespace cannot be null for listing syncsets")
	}
	if !isSyncSet && namespace != "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "namespace should be null for listing selectorsyncsets")
	}

	ss, err := f.hiveSyncSetManager.List(ctx, namespace, label, ssType)
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
