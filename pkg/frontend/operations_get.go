package frontend

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
)

func (f *frontend) getOperations(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(contextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	_, found := api.APIs[api.APIVersionType{APIVersion: r.URL.Query().Get("api-version"), Type: "OpenShiftCluster"}]
	if !found {
		api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeInvalidResourceNamespace, "", "The resource namespace '%s' could not be found for api version '%s'.", vars["resourceProviderNamespace"], r.URL.Query().Get("api-version"))
		return
	}

	ops := &api.OperationList{
		Value: []api.Operation{
			{
				Name: "Microsoft.RedHatOpenShift/openShiftClusters/read",
				Display: api.Display{
					Provider:  "Azure Red Hat OpenShift",
					Resource:  "openShiftClusters",
					Operation: "Read OpenShift cluster",
				},
			},
			{
				Name: "Microsoft.RedHatOpenShift/openShiftClusters/write",
				Display: api.Display{
					Provider:  "Azure Red Hat OpenShift",
					Resource:  "openShiftClusters",
					Operation: "Write OpenShift cluster",
				},
			},
			{
				Name: "Microsoft.RedHatOpenShift/openShiftClusters/delete",
				Display: api.Display{
					Provider:  "Azure Red Hat OpenShift",
					Resource:  "openShiftClusters",
					Operation: "Delete OpenShift cluster",
				},
			},
			{
				Name: "Microsoft.RedHatOpenShift/openShiftClusters/credentials/action",
				Display: api.Display{
					Provider:  "Azure Red Hat OpenShift",
					Resource:  "openShiftClusters/credentials",
					Operation: "Gets credentials of a OpenShift cluster",
				},
			},
		},
	}

	b, err := json.MarshalIndent(ops, "", "  ")
	if err != nil {
		log.Error(err)
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
		return
	}

	w.Write(b)
	w.Write([]byte{'\n'})
}
