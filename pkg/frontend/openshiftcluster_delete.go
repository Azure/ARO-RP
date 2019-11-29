package frontend

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
)

type noContent struct{}

func (noContent) Error() string { return "" }

func (f *frontend) deleteOpenShiftCluster(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(contextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	_, found := api.APIs[api.APIVersionType{APIVersion: r.URL.Query().Get("api-version"), Type: "OpenShiftCluster"}]
	if !found {
		api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeInvalidResourceType, "", "The resource type '%s' could not be found in the namespace '%s' for api version '%s'.", vars["resourceType"], vars["resourceProviderNamespace"], r.URL.Query().Get("api-version"))
		return
	}

	_, err := f.db.OpenShiftClusters.Patch(api.Key(r.URL.Path), func(doc *api.OpenShiftClusterDocument) error {
		return f._deleteOpenShiftCluster(&request{
			resourceID: r.URL.Path,
		}, doc)
	})
	if err != nil {
		switch err := err.(type) {
		case *api.CloudError:
			api.WriteCloudError(w, err)
		case *noContent:
			w.WriteHeader(http.StatusNoContent)
		default:
			log.Error(err)
			api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
		}
		return
	}
}

func (f *frontend) _deleteOpenShiftCluster(r *request, doc *api.OpenShiftClusterDocument) error {
	if doc == nil {
		return &noContent{}
	}

	_, err := f.validateSubscriptionState(doc.Key, api.SubscriptionStateRegistered, api.SubscriptionStateWarned, api.SubscriptionStateSuspended)
	if err != nil {
		return err
	}

	err = validateTerminalProvisioningState(doc.OpenShiftCluster.Properties.ProvisioningState)
	if err != nil {
		return err
	}

	doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateDeleting
	doc.Dequeues = 0

	return nil
}
