package frontend

import (
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"

	"github.com/jim-minter/rp/pkg/api"
)

var rxResourceGroupName = regexp.MustCompile(`(?i)^[-a-z0-9_().]{0,89}[-a-z0-9_()]$`)

func (f *frontend) isValidRequestPath(w http.ResponseWriter, r *http.Request) bool {
	vars := mux.Vars(r)

	if _, found := vars["subscriptionId"]; found {
		_, err := uuid.FromString(vars["subscriptionId"])
		if err != nil {
			api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeInvalidSubscriptionID, "", "The provided subscription identifier '%s' is malformed or invalid.", vars["subscriptionId"])
			return false
		}
	}

	if _, found := vars["resourceGroupName"]; found {
		if !rxResourceGroupName.MatchString(vars["resourceGroupName"]) {
			api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeResourceGroupNotFound, "", "Resource group '%s' could not be found.", vars["resourceGroupName"])
			return false
		}
	}

	if _, found := vars["resourceProviderNamespace"]; found {
		if vars["resourceProviderNamespace"] != resourceProviderNamespace {
			api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeInvalidResourceNamespace, "", "The resource namespace '%s' is invalid.", vars["resourceProviderNamespace"])
			return false
		}
	}

	if _, found := vars["resourceType"]; found {
		if vars["resourceType"] != resourceType {
			api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeInvalidResourceType, "", "The resource type '%s' could not be found in the namespace '%s' for api version '%s'.", vars["resourceType"], vars["resourceProviderNamespace"], r.URL.Query().Get("api-version"))
			return false
		}
	}

	if _, found := vars["resourceName"]; found {
		// TODO: if we continue to use this as a prefix we will need to shorten the validation here
		if !rxResourceGroupName.MatchString(vars["resourceName"]) {
			api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s/%s' under resource group '%s' was not found.", vars["resourceProviderNamespace"], vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
			return false
		}
	}

	return true
}
