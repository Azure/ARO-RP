package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
)

type OCMValidator struct {
	ValidOCMClientIDs []string
}

func (o OCMValidator) ValidateOCMClient(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ocmResourceType := chi.URLParam(r, "ocmResourceType"); ocmResourceType != "" {
			if valid, err := validateOCMFromSystemData(r, o.ValidOCMClientIDs); !valid || err != nil {
				_log, ok := r.Context().Value(ContextKeyLog).(*logrus.Entry)
				if ok {
					_log.Error("failed to validate ocm clientId", err)
				}
				api.WriteError(w, http.StatusForbidden, api.CloudErrorCodeForbidden, "", "Forbidden.")
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

func validateOCMFromSystemData(r *http.Request, validClientIDs []string) (bool, error) {
	systemDataHeaderStr := r.Header.Get(ArmSystemDataHeaderKey)
	var systemData *api.SystemData
	err := json.Unmarshal([]byte(systemDataHeaderStr), &systemData)
	if err != nil {
		return false, err
	}

	if systemData.LastModifiedByType != api.CreatedByTypeApplication {
		return false, fmt.Errorf("only applications are authorized, received: %q", systemData.LastModifiedByType)
	}

	for _, validClientId := range validClientIDs {
		if strings.EqualFold(systemData.LastModifiedBy, validClientId) {
			return true, nil
		}
	}
	return false, errors.New("invalid ocm clientID")
}
