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
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

const (
	resourceProviderNamespace = "Microsoft.RedHatOpenShift"
	resourceType              = "openShiftClusters"
)

var rxResourceGroupName = regexp.MustCompile(`^[-a-z0-9_().]{0,89}[-a-z0-9_()]$`)

type ValidateMiddleware struct {
	Location string
	Apis     map[string]*api.Version
}

func (v ValidateMiddleware) Validate(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		route := mux.CurrentRoute(r)

		apiVersion := r.URL.Query().Get(api.APIVersionKey)

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
				api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceType, "", "The resource type '%s' could not be found in the namespace '%s' for api version '%s'.", vars["resourceType"], vars["resourceProviderNamespace"], apiVersion)
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
			if !strings.EqualFold(vars["location"], v.Location) {
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
			if query == "api-version=" {
				hasVariableAPIVersion = true
				break
			}
		}

		_, apiVersionExists := v.Apis[apiVersion]
		if (err != nil || hasVariableAPIVersion) && apiVersion != "" && !apiVersionExists {
			api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidResourceType, "", "The resource type '%s' could not be found in the namespace '%s' for api version '%s'.", vars["resourceType"], vars["resourceProviderNamespace"], apiVersion)
			return
		}

		h.ServeHTTP(w, r)
	})
}
