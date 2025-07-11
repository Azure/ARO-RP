package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/sirupsen/logrus"

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
	if f.hiveSyncSetManager == nil {
		return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "hive is not enabled")
	}

	if isSyncSet && namespace == "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "namespace cannot be null for listing syncsets")
	}
	if !isSyncSet && namespace != "" {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "namespace should be null for listing selectorsyncsets")
	}

	var b []byte
	if isSyncSet {
		items, err := f.hiveSyncSetManager.ListSyncSets(ctx, namespace, label)
		if err != nil {
			return nil, err
		}
		type minimalMeta struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		}
		type minimalSyncSet struct {
			Metadata minimalMeta `json:"metadata"`
		}
		var result []minimalSyncSet
		for _, ss := range items {
			result = append(result, minimalSyncSet{
				Metadata: minimalMeta{
					Name:      ss.Name,
					Namespace: ss.Namespace,
				},
			})
		}
		b, err = json.Marshal(struct {
			Items interface{} `json:"items"`
		}{Items: result})
		if err != nil {
			return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "unable to marshal response")
		}
	} else {
		items, err := f.hiveSyncSetManager.ListSelectorSyncSets(ctx, namespace, label)
		if err != nil {
			return nil, err
		}
		type minimalMeta struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		}
		type minimalSelectorSyncSet struct {
			Metadata minimalMeta `json:"metadata"`
		}
		var result []minimalSelectorSyncSet
		for _, ss := range items {
			result = append(result, minimalSelectorSyncSet{
				Metadata: minimalMeta{
					Name:      ss.Name,
					Namespace: ss.Namespace,
				},
			})
		}
		b, err = json.Marshal(struct {
			Items interface{} `json:"items"`
		}{Items: result})
		if err != nil {
			return nil, api.NewCloudError(http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "unable to marshal response")
		}
	}
	return b, nil
}
