package frontend

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
)

func (f *frontend) postOpenShiftClusterCredentials(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(contextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	var err error
	r, err = readBody(w, r)
	if err != nil {
		api.WriteCloudError(w, err.(*api.CloudError))
		return
	}

	body := r.Context().Value(contextKeyBody).([]byte)
	if !json.Valid(body) {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
		return
	}

	r.URL.Path = filepath.Dir(r.URL.Path)

	b, err := f._getOpenShiftCluster(r, api.APIs[vars["api-version"]]["OpenShiftClusterCredentials"].(api.OpenShiftClusterToExternal))

	reply(log, w, b, err)
}
