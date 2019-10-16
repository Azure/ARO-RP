package frontend

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
)

func (f *frontend) getOpenShiftCluster(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(contextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	toExternal, found := api.APIs[api.APIVersionType{APIVersion: r.URL.Query().Get("api-version"), Type: "OpenShiftCluster"}]
	if !found {
		f.error(w, http.StatusNotFound, api.CloudErrorCodeInvalidResourceType, "", "The resource type '%s' could not be found in the namespace '%s' for api version '%s'.", vars["resourceType"], vars["resourceProviderNamespace"], r.URL.Query().Get("api-version"))
		return
	}

	b, err := f._getOpenShiftCluster(&request{
		resourceID:        r.URL.Path,
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
	if err != nil {
		return nil, err
	}

	if doc == nil {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s' under resource group '%s' was not found.", r.resourceType, r.resourceName, r.resourceGroupName)
	}

	doc.OpenShiftCluster.ID = r.resourceID
	doc.OpenShiftCluster.Name = r.resourceName
	doc.OpenShiftCluster.Type = r.resourceType
	doc.OpenShiftCluster.Properties.PullSecret = nil

	return json.MarshalIndent(r.toExternal(doc.OpenShiftCluster), "", "  ")
}

func (f *frontend) getOpenShiftClusters(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(contextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	toExternal, found := api.APIs[api.APIVersionType{APIVersion: r.URL.Query().Get("api-version"), Type: "OpenShiftCluster"}]
	if !found {
		f.error(w, http.StatusNotFound, api.CloudErrorCodeInvalidResourceType, "", "The resource type '%s' could not be found in the namespace '%s' for api version '%s'.", vars["resourceType"], vars["resourceProviderNamespace"], r.URL.Query().Get("api-version"))
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

func (f *frontend) _getOpenShiftClusters(r *request) ([]byte, error) {
	prefix := "/subscriptions/" + r.subscriptionID + "/"
	if r.resourceGroupName != "" {
		prefix += "resourceGroups/" + r.resourceGroupName + "/"
	}
	i := f.db.ListByPrefix(r.subscriptionID, prefix)

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
			doc.OpenShiftCluster.Type = r.resourceType
			doc.OpenShiftCluster.Properties.PullSecret = nil
			rv.Value = append(rv.Value, r.toExternal(doc.OpenShiftCluster))
		}
	}

	return json.MarshalIndent(rv, "", "  ")
}

func (f *frontend) getOpenShiftClusterCredentials(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(contextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	toExternal, found := api.APIs[api.APIVersionType{APIVersion: r.URL.Query().Get("api-version"), Type: "OpenShiftClusterCredentials"}]
	if !found {
		f.error(w, http.StatusNotFound, api.CloudErrorCodeInvalidResourceType, "The resource type '%s' could not be found in the namespace '%s' for api version '%s'.", vars["resourceType"], vars["resourceProviderNamespace"], r.URL.Query().Get("api-version"))
		return
	}

	b, err := f._getOpenShiftClusterCredentials(&request{
		resourceID:        filepath.Dir(r.URL.Path),
		resourceGroupName: vars["resourceGroupName"],
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

func (f *frontend) _getOpenShiftClusterCredentials(r *request) ([]byte, error) {
	doc, err := f.db.Get(r.resourceID)
	if err != nil {
		return nil, err
	}

	if doc == nil {
		return nil, api.NewCloudError(http.StatusNotFound, api.CloudErrorCodeResourceNotFound, "", "The Resource '%s/%s/credentials' under resource group '%s' was not found.", r.resourceType, r.resourceName, r.resourceGroupName)
	}

	return json.MarshalIndent(r.toExternal(doc.OpenShiftCluster), "", "  ")
}
