package middleware

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"

	"github.com/jim-minter/rp/pkg/api"
)

const (
	resourceProviderNamespace = "Microsoft.RedHatOpenShift"
	resourceType              = "openShiftClusters"
)

var rxResourceGroupName = regexp.MustCompile(`^[-a-z0-9_().]{0,89}[-a-z0-9_()]$`)

func Validate(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		route := mux.CurrentRoute(r)

		if route == nil {
			api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
			return
		}

		if _, found := vars["subscriptionId"]; found {
			_, err := uuid.FromString(vars["subscriptionId"])
			if err != nil || vars["subscriptionId"] != strings.ToLower(vars["subscriptionId"]) {
				api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeInvalidSubscriptionID, "", "The provided subscription identifier '%s' is malformed or invalid.", vars["subscriptionId"])
				return
			}
		}

		if _, found := vars["resourceGroupName"]; found {
			if !rxResourceGroupName.MatchString(vars["resourceGroupName"]) {
				api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeResourceGroupNotFound, "", "Resource group '%s' could not be found.", vars["resourceGroupName"])
				return
			}
		}

		if _, found := vars["resourceProviderNamespace"]; found {
			if vars["resourceProviderNamespace"] != strings.ToLower(resourceProviderNamespace) {
				api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeInvalidResourceNamespace, "", "The resource namespace '%s' is invalid.", vars["resourceProviderNamespace"])
				return
			}
		}

		if _, found := vars["resourceType"]; found {
			if vars["resourceType"] != strings.ToLower(resourceType) {
				api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeInvalidResourceType, "", "The resource type '%s' could not be found in the namespace '%s' for api version '%s'.", vars["resourceType"], vars["resourceProviderNamespace"], vars["api-version"])
				return
			}
		}

		if _, found := vars["resourceName"]; found {
			if !rxResourceGroupName.MatchString(vars["resourceName"]) {
				api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s/%s' under resource group '%s' was not found.", vars["resourceProviderNamespace"], vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
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
				if _, found := api.APIs[vars["api-version"]]; !found {
					api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeInvalidResourceType, "", "The resource type '%s' could not be found in the namespace '%s' for api version '%s'.", vars["resourceType"], vars["resourceProviderNamespace"], vars["api-version"])
					return
				}
			}
		}

		h.ServeHTTP(w, r)
	})
}
