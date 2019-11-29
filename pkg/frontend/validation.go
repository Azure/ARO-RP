package frontend

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database/cosmosdb"
)

var rxResourceGroupName = regexp.MustCompile(`^[-a-z0-9_().]{0,89}[-a-z0-9_()]$`)

func (f *frontend) isValidRequestPath(w http.ResponseWriter, r *http.Request) bool {
	vars := mux.Vars(r)

	if _, found := vars["subscriptionId"]; found {
		_, err := uuid.FromString(vars["subscriptionId"])
		if err != nil || vars["subscriptionId"] != strings.ToLower(vars["subscriptionId"]) {
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
		if vars["resourceProviderNamespace"] != strings.ToLower(resourceProviderNamespace) {
			api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeInvalidResourceNamespace, "", "The resource namespace '%s' is invalid.", vars["resourceProviderNamespace"])
			return false
		}
	}

	if _, found := vars["resourceType"]; found {
		if vars["resourceType"] != strings.ToLower(resourceType) {
			api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeInvalidResourceType, "", "The resource type '%s' could not be found in the namespace '%s' for api version '%s'.", vars["resourceType"], vars["resourceProviderNamespace"], r.URL.Query().Get("api-version"))
			return false
		}
	}

	if _, found := vars["resourceName"]; found {
		if !rxResourceGroupName.MatchString(vars["resourceName"]) {
			api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s/%s' under resource group '%s' was not found.", vars["resourceProviderNamespace"], vars["resourceType"], vars["resourceName"], vars["resourceGroupName"])
			return false
		}
	}

	return true
}

func validateTerminalProvisioningState(state api.ProvisioningState) error {
	switch state {
	case api.ProvisioningStateSucceeded, api.ProvisioningStateFailed:
		return nil
	}

	return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeRequestNotAllowed, "", "Request is not allowed in provisioningState '%s'.", state)
}

func (f *frontend) validateSubscriptionState(key api.Key, allowedStates ...api.SubscriptionState) (*api.SubscriptionDocument, error) {
	r, err := azure.ParseResourceID(string(key))
	if err != nil {
		return nil, err
	}

	doc, err := f.db.Subscriptions.Get(api.Key(r.SubscriptionID))
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidSubscriptionState, "", "Request is not allowed in unregistered subscription '%s'.", r.SubscriptionID)
	case err != nil:
		return nil, err
	}

	for _, allowedState := range allowedStates {
		if doc.Subscription.State == allowedState {
			return doc, nil
		}
	}

	return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidSubscriptionState, "", "Request is not allowed in subscription in state '%s'.", doc.Subscription.State)
}
