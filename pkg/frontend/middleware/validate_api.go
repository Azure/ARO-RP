package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Azure/ARO-RP/pkg/api"
)

type ApiVersionValidator struct {
	APIs map[string]*api.Version
}

func (a ApiVersionValidator) ValidateAPIVersion(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiVersion := r.URL.Query().Get(api.APIVersionKey)
		resourceType := chi.URLParam(r, "resourceType")
		resourceProviderNamespace := chi.URLParam(r, "resourceProviderNamespace")
		if _, apiVersionExists := a.APIs[apiVersion]; !apiVersionExists {
			api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceType, "", fmt.Sprintf("The resource type '%s' could not be found in the namespace '%s' for api version '%s'.", resourceType, resourceProviderNamespace, apiVersion))
			return
		}

		h.ServeHTTP(w, r)
	})
}

func (a ApiVersionValidator) ValidatePreflightAPIVersion(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiVersion := r.URL.Query().Get(api.APIVersionKey)
		resourceProviderNamespace := chi.URLParam(r, "resourceProviderNamespace")
		if _, apiVersionExists := a.APIs[apiVersion]; !apiVersionExists {
			api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceType, "", fmt.Sprintf("Api version %q is invalid for preflight validation in resource provider %q.", apiVersion, resourceProviderNamespace))
			return
		}

		h.ServeHTTP(w, r)
	})
}
