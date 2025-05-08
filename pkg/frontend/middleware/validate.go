package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

const (
	wantedResourceProviderNamespace = "Microsoft.RedHatOpenShift"
	resourceTypeOpenshiftCluster    = "openShiftClusters"
)

var rxResourceGroupName = regexp.MustCompile(`^[-a-z0-9_().]{0,89}[-a-z0-9_()]$`)

type ValidateMiddleware struct {
	Location string
	Apis     map[string]*api.Version
}

func (v ValidateMiddleware) Validate(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		subId := chi.URLParam(r, "subscriptionId")
		resourceGroupName := chi.URLParam(r, "resourceGroupName")
		resourceProviderNamespace := chi.URLParam(r, "resourceProviderNamespace")
		resourceType := chi.URLParam(r, "resourceType")
		location := chi.URLParam(r, "location")
		operationId := chi.URLParam(r, "operationId")
		resourceName := chi.URLParam(r, "resourceName")

		apiVersion := r.URL.Query().Get(api.APIVersionKey)

		if r.URL.Path != strings.ToLower(r.URL.Path) {
			if log, ok := r.Context().Value(ContextKeyLog).(*logrus.Entry); ok {
				log.Error("path was not lower case")
			}
			api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
			return
		}

		if subId != "" {
			valid := uuid.IsValid(subId)
			if !valid {
				api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidSubscriptionID, "", fmt.Sprintf("The provided subscription identifier '%s' is malformed or invalid.", subId))
				return
			}
		}

		if resourceGroupName != "" {
			if !rxResourceGroupName.MatchString(resourceGroupName) {
				api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeResourceGroupNotFound, "", fmt.Sprintf("Resource group '%s' could not be found.", resourceGroupName))
				return
			}
		}

		if resourceProviderNamespace != "" {
			if resourceProviderNamespace != strings.ToLower(wantedResourceProviderNamespace) {
				api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceNamespace, "", fmt.Sprintf("The resource namespace '%s' is invalid.", resourceProviderNamespace))
				return
			}
		}

		if resourceType != "" {
			if resourceType != strings.ToLower(resourceTypeOpenshiftCluster) {
				api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceType, "", fmt.Sprintf("The resource type '%s' could not be found in the namespace '%s' for api version '%s'.", resourceType, resourceProviderNamespace, apiVersion))
				return
			}
		}

		if resourceName != "" {
			if !rxResourceGroupName.MatchString(resourceName) {
				api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeResourceNotFound, "", fmt.Sprintf("The Resource '%s/%s/%s' under resource group '%s' was not found.", resourceProviderNamespace, resourceType, resourceName, resourceGroupName))
				return
			}
		}

		if location != "" {
			if !strings.EqualFold(location, v.Location) {
				api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidLocation, "", fmt.Sprintf("The provided location '%s' is malformed or invalid.", location))
				return
			}
		}

		if operationId != "" {
			valid := uuid.IsValid(operationId)
			if !valid {
				api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidOperationID, "", fmt.Sprintf("The provided operation identifier '%s' is malformed or invalid.", operationId))
				return
			}
		}

		h.ServeHTTP(w, r)
	})
}
