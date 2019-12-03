package frontend

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
)

func (f *frontend) getOperations(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(contextKeyLog).(*logrus.Entry)

	l := &api.OperationList{
		Operations: []api.Operation{
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

	b, err := json.MarshalIndent(l, "", "  ")
	reply(log, w, b, err)
}
