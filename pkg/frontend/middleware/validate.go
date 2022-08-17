package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

const (
	resourceProviderNamespace = "Microsoft.RedHatOpenShift"
	resourceType              = "openShiftClusters"
)

var rxResourceGroupName = regexp.MustCompile(`^[-a-z0-9_().]{0,89}[-a-z0-9_()]$`)

func Validate(env env.Core, apis map[string]*api.Version) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			route := mux.CurrentRoute(r)

			if route == nil {
				if log, ok := r.Context().Value(ContextKeyLog).(*logrus.Entry); ok {
					log.Error("route was nil")
				}
				api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
				return
			}

			if r.URL.Path != strings.ToLower(r.URL.Path) {
				if log, ok := r.Context().Value(ContextKeyLog).(*logrus.Entry); ok {
					log.Error("path was not lower case")
				}
				api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
				return
			}

			if _, found := vars["subscriptionId"]; found {
				valid := uuid.IsValid(vars["subscriptionId"])
				if !valid {
					api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidSubscriptionID, "", "The provided subscription identifier '%s' is malformed or invalid.", vars["subscriptionId"])
					return
				}
			}

			if _, found := vars["resourceGroupName"]; found {
				if !rxResourceGroupName.MatchString(vars["resourceGroupName"]) {
					api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeResourceGroupNotFound, "", "Resource group '%s' could not be found.", vars["resourceGroupName"])
					return
				}
			}

			if _, found := vars["resourceProviderNamespace"]; found {
				if vars["resourceProviderNamespace"] != strings.ToLower(resourceProviderNamespace) {
					api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceNamespace, "", "The resource namespace '%s' is invalid.", vars["resourceProviderNamespace"])
					return
				}
			}

			if _, found := vars["resourceType"]; found {
				if vars["resourceType"] != strings.ToLower(resourceType) {
					api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceType, "", "The resource type '%s' could not be found in the namespace '%s' for api version '%s'.", vars["resourceType"], vars["resourceProviderNamespace"], vars["api-version"])
					return
				}
			}

			if _, found := vars["resourceName"]; found {
				if !rxResourceGroupName.MatchString(vars["resourceName"]) {
					api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s/%s' under resource group '%s' was not found.", vars["resourceProviderNamespace"], vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
					return
				}
			}

			if _, found := vars["location"]; found {
				if !strings.EqualFold(vars["location"], env.Location()) {
					api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidLocation, "", "The provided location '%s' is malformed or invalid.", vars["location"])
					return
				}
			}

			if _, found := vars["operationId"]; found {
				valid := uuid.IsValid(vars["operationId"])
				if !valid {
					api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidOperationID, "", "The provided operation identifier '%s' is malformed or invalid.", vars["operationId"])
					return
				}
			}

			queries, err := route.GetQueriesTemplates()
			var hasVariableAPIVersion bool
			for _, query := range queries {
				if strings.HasPrefix(query, "api-version=") && strings.ContainsRune(query, '{') {
					hasVariableAPIVersion = true
					break
				}
			}

			if err != nil || hasVariableAPIVersion {
				if _, found := vars["api-version"]; found {
					if _, found := apis[vars["api-version"]]; !found {
						api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceType, "", "The resource type '%s' could not be found in the namespace '%s' for api version '%s'.", vars["resourceType"], vars["resourceProviderNamespace"], vars["api-version"])
						return
					}
				}
			}

			h.ServeHTTP(w, r)
		})
	}
}
