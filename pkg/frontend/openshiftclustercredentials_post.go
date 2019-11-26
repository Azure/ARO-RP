package frontend

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/jim-minter/rp/pkg/api"
)

func (f *frontend) postOpenShiftClusterCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		api.WriteError(w, http.StatusUnsupportedMediaType, api.CloudErrorCodeUnsupportedMediaType, "", "The content media type '%s' is not supported. Only 'application/json' is supported.", r.Header.Get("Content-Type"))
		return
	}

	body, err := ioutil.ReadAll(http.MaxBytesReader(w, r.Body, 1048576))
	if err != nil {
		api.WriteError(w, http.StatusUnsupportedMediaType, api.CloudErrorCodeInvalidResource, "", "The resource definition is invalid.")
		return
	}

	if !json.Valid(body) {
		api.WriteError(w, http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
		return
	}

	f.get(w, r, filepath.Dir(r.URL.Path), "OpenShiftClusterCredentials")
}
