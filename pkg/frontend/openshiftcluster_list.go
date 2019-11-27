package frontend

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
)

func (f *frontend) getOpenShiftClusters(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(contextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	toExternal, found := api.APIs[api.APIVersionType{APIVersion: r.URL.Query().Get("api-version"), Type: "OpenShiftCluster"}]
	if !found {
		api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeInvalidResourceType, "", "The resource type '%s' could not be found in the namespace '%s' for api version '%s'.", vars["resourceType"], vars["resourceProviderNamespace"], r.URL.Query().Get("api-version"))
		return
	}

	b, err := f._getOpenShiftClusters(&request{
		subscriptionID:    vars["subscriptionId"],
		resourceGroupName: vars["resourceGroupName"],
		resourceType:      vars["resourceProviderNamespace"] + "/" + vars["resourceType"],
		toExternal:        toExternal,
	})
	if err != nil {
		switch err := err.(type) {
		case *api.CloudError:
			api.WriteCloudError(w, err)
		default:
			log.Error(err)
			api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
		}
		return
	}

	w.Write(b)
	w.Write([]byte{'\n'})
}

func (f *frontend) _getOpenShiftClusters(r *request) ([]byte, error) {
	prefix := "/subscriptions/" + r.subscriptionID + "/"
	if r.resourceGroupName != "" {
		prefix += "resourcegroups/" + r.resourceGroupName + "/"
	}

	i, err := f.db.ListByPrefix(r.subscriptionID, api.Key(prefix))
	if err != nil {
		return nil, err
	}

	var rv struct {
		Value []api.External `json:"value"`
	}

	for {
		docs, err := i.Next()
		if err != nil {
			return nil, err
		}
		if docs == nil {
			break
		}

		for _, doc := range docs.OpenShiftClusterDocuments {
			doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret = ""
			rv.Value = append(rv.Value, r.toExternal(doc.OpenShiftCluster))
		}
	}

	return json.MarshalIndent(rv, "", "  ")
}
