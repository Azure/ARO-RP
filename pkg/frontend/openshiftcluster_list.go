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

	b, err := f._getOpenShiftClusters(&request{
		subscriptionID:    vars["subscriptionId"],
		resourceGroupName: vars["resourceGroupName"],
		resourceType:      vars["resourceProviderNamespace"] + "/" + vars["resourceType"],
		toExternals:       api.APIs[vars["api-version"]]["OpenShiftCluster"].(api.OpenShiftClustersToExternal),
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

	i, err := f.db.OpenShiftClusters.ListByPrefix(r.subscriptionID, api.Key(prefix))
	if err != nil {
		return nil, err
	}

	var ocs []*api.OpenShiftCluster

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
			ocs = append(ocs, doc.OpenShiftCluster)
		}
	}

	return json.MarshalIndent(r.toExternals.OpenShiftClustersToExternal(ocs), "", "  ")
}
