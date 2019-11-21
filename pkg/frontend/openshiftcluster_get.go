package frontend

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database/cosmosdb"
)

func (f *frontend) getOpenShiftCluster(w http.ResponseWriter, r *http.Request) {
	f.get(w, r, r.URL.Path, "OpenShiftCluster")
}

func (f *frontend) get(w http.ResponseWriter, r *http.Request, resourceID, typ string) {
	log := r.Context().Value(contextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	toExternal, found := api.APIs[api.APIVersionType{APIVersion: r.URL.Query().Get("api-version"), Type: typ}]
	if !found {
		f.error(w, http.StatusNotFound, api.CloudErrorCodeInvalidResourceType, "", "The resource type '%s' could not be found in the namespace '%s' for api version '%s'.", vars["resourceType"], vars["resourceProviderNamespace"], r.URL.Query().Get("api-version"))
		return
	}

	b, err := f._getOpenShiftCluster(&request{
		resourceID:        resourceID,
		resourceGroupName: vars["resourceGroupName"],
		resourceName:      vars["resourceName"],
		resourceType:      vars["resourceProviderNamespace"] + "/" + vars["resourceType"],
		toExternal:        toExternal,
	})
	if err != nil {
		switch err := err.(type) {
		case *api.CloudError:
			f.cloudError(w, err)
		default:
			log.Error(err)
			f.error(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
		}
		return
	}

	w.Write(b)
	w.Write([]byte{'\n'})
}

func (f *frontend) _getOpenShiftCluster(r *request) ([]byte, error) {
	doc, err := f.db.Get(r.resourceID)
	switch {
	case cosmosdb.IsErrorStatusCode(err, http.StatusNotFound):
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", r.resourceType, r.resourceName, r.resourceGroupName)
	case err != nil:
		return nil, err
	}

	doc.OpenShiftCluster.ID = r.resourceID
	doc.OpenShiftCluster.Name = r.resourceName
	doc.OpenShiftCluster.Type = r.resourceType
	doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret = ""

	return json.MarshalIndent(r.toExternal(doc.OpenShiftCluster), "", "  ")
}
