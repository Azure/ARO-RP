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

	b, err := f._getOpenShiftClusters(r, api.APIs[vars["api-version"]]["OpenShiftCluster"].(api.OpenShiftClustersToExternal))

	reply(log, w, b, err)
}

func (f *frontend) _getOpenShiftClusters(r *http.Request, externals api.OpenShiftClustersToExternal) ([]byte, error) {
	vars := mux.Vars(r)

	prefix := "/subscriptions/" + vars["subscriptionId"] + "/"
	if vars["resourceGroupName"] != "" {
		prefix += "resourcegroups/" + vars["resourceGroupName"] + "/"
	}

	i, err := f.db.OpenShiftClusters.ListByPrefix(vars["subscriptionId"], api.Key(prefix))
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

	return json.MarshalIndent(externals.OpenShiftClustersToExternal(ocs), "", "    ")
}
