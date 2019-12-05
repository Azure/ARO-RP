package frontend

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database/cosmosdb"
)

func (f *frontend) putOrPatchOpenShiftCluster(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(contextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	var err error
	r, err = readBody(w, r)
	if err != nil {
		api.WriteCloudError(w, err.(*api.CloudError))
		return
	}

	var b []byte
	var created bool
	err = cosmosdb.RetryOnPreconditionFailed(func() error {
		b, created, err = f._putOrPatchOpenShiftCluster(r, api.APIs[vars["api-version"]]["OpenShiftCluster"].(api.OpenShiftClusterToInternal), api.APIs[vars["api-version"]]["OpenShiftCluster"].(api.OpenShiftClusterToExternal))
		return err
	})
	if err == nil && created {
		w.WriteHeader(http.StatusCreated)
	}

	reply(log, w, b, err)
}

func (f *frontend) _putOrPatchOpenShiftCluster(r *http.Request, internal api.OpenShiftClusterToInternal, external api.OpenShiftClusterToExternal) ([]byte, bool, error) {
	vars := mux.Vars(r)
	body := r.Context().Value(contextKeyBody).([]byte)

	subdoc, err := f.validateSubscriptionState(api.Key(r.URL.Path), api.SubscriptionStateRegistered)
	if err != nil {
		return nil, false, err
	}

	doc, err := f.db.OpenShiftClusters.Get(api.Key(r.URL.Path))
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, false, err
	}

	isCreate := doc == nil

	if isCreate {
		originalPath := r.Context().Value(contextKeyOriginalPath).(string)
		originalR, err := azure.ParseResourceID(originalPath)
		if err != nil {
			return nil, false, err
		}

		doc = &api.OpenShiftClusterDocument{
			ID:  uuid.NewV4().String(),
			Key: api.Key(r.URL.Path),
			OpenShiftCluster: &api.OpenShiftCluster{
				ID:   originalPath,
				Name: originalR.ResourceName,
				Type: fmt.Sprintf("%s/%s", originalR.Provider, originalR.ResourceType),
				Properties: api.Properties{
					ProvisioningState: api.ProvisioningStateSucceeded,
					// TODO: ResourceGroup should be exposed in external API
					ResourceGroup: vars["resourceName"],
					ServicePrincipalProfile: api.ServicePrincipalProfile{
						TenantID: subdoc.Subscription.Properties.TenantID,
					},
				},
			},
		}
	}

	err = validateTerminalProvisioningState(doc.OpenShiftCluster.Properties.ProvisioningState)
	if err != nil {
		return nil, false, err
	}

	if doc.OpenShiftCluster.Properties.ProvisioningState == api.ProvisioningStateFailed {
		switch doc.OpenShiftCluster.Properties.FailedProvisioningState {
		case api.ProvisioningStateCreating:
			return nil, false, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeRequestNotAllowed, "", "Request is not allowed on cluster whose creation failed. Delete the cluster.")
		case api.ProvisioningStateUpdating:
			// allow
		case api.ProvisioningStateDeleting:
			return nil, false, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeRequestNotAllowed, "", "Request is not allowed on cluster whose deletion failed. Delete the cluster.")
		default:
			return nil, false, fmt.Errorf("unexpected failedProvisioningState %q", doc.OpenShiftCluster.Properties.FailedProvisioningState)
		}
	}

	var ext interface{}
	switch r.Method {
	case http.MethodPut:
		ext = external.OpenShiftClusterToExternal(&api.OpenShiftCluster{
			ID:   doc.OpenShiftCluster.ID,
			Name: doc.OpenShiftCluster.Name,
			Type: doc.OpenShiftCluster.Type,
			Properties: api.Properties{
				ProvisioningState: doc.OpenShiftCluster.Properties.ProvisioningState,
			},
		})

	case http.MethodPatch:
		ext = external.OpenShiftClusterToExternal(doc.OpenShiftCluster)
	}

	err = json.Unmarshal(body, &ext)
	if err != nil {
		return nil, false, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
	}

	if isCreate {
		err = internal.ValidateOpenShiftCluster(f.env.Location(), r.URL.Path, ext, nil)
	} else {
		err = internal.ValidateOpenShiftCluster(f.env.Location(), r.URL.Path, ext, doc.OpenShiftCluster)
	}
	if err != nil {
		return nil, false, err
	}

	oldID, oldName, oldType := doc.OpenShiftCluster.ID, doc.OpenShiftCluster.Name, doc.OpenShiftCluster.Type
	internal.OpenShiftClusterToInternal(ext, doc.OpenShiftCluster)
	doc.OpenShiftCluster.ID, doc.OpenShiftCluster.Name, doc.OpenShiftCluster.Type = oldID, oldName, oldType

	if isCreate {
		doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateCreating
	} else {
		doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateUpdating
		doc.Dequeues = 0
	}

	err = internal.ValidateOpenShiftClusterDynamic(r.Context(), f.fpAuthorizer, doc.OpenShiftCluster)
	if err != nil {
		return nil, false, err
	}

	if isCreate {
		doc, err = f.db.OpenShiftClusters.Create(doc)
	} else {
		doc, err = f.db.OpenShiftClusters.Update(doc)
	}
	if err != nil {
		return nil, false, err
	}

	doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret = ""

	b, err := json.MarshalIndent(external.OpenShiftClusterToExternal(doc.OpenShiftCluster), "", "    ")
	if err != nil {
		return nil, false, err
	}

	return b, isCreate, nil
}
