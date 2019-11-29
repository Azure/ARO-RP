package frontend

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
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

	toExternal, found := api.APIs[api.APIVersionType{APIVersion: r.URL.Query().Get("api-version"), Type: "OpenShiftCluster"}]
	if !found {
		api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeInvalidResourceType, "", "The resource type '%s' could not be found in the namespace '%s' for api version '%s'.", vars["resourceType"], vars["resourceProviderNamespace"], r.URL.Query().Get("api-version"))
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		api.WriteError(w, http.StatusUnsupportedMediaType, api.CloudErrorCodeUnsupportedMediaType, "", "The content media type '%s' is not supported. Only 'application/json' is supported.", r.Header.Get("Content-Type"))
		return
	}

	body, err := ioutil.ReadAll(http.MaxBytesReader(w, r.Body, 1048576))
	if err != nil {
		api.WriteError(w, http.StatusUnsupportedMediaType, api.CloudErrorCodeInvalidResource, "", "The resource definition is invalid.")
		return
	}

	var b []byte
	var created bool
	err = cosmosdb.RetryOnPreconditionFailed(func() error {
		b, created, err = f._putOrPatchOpenShiftCluster(&request{
			context:      r.Context(),
			method:       r.Method,
			resourceID:   r.URL.Path,
			resourceName: vars["resourceName"],
			resourceType: vars["resourceProviderNamespace"] + "/" + vars["resourceType"],
			body:         body,
			toExternal:   toExternal,
		})
		return err
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

	if created {
		w.WriteHeader(http.StatusCreated)
	}
	w.Write(b)
	w.Write([]byte{'\n'})
}

func (f *frontend) _putOrPatchOpenShiftCluster(r *request) ([]byte, bool, error) {
	originalPath := r.context.Value(contextKeyOriginalPath).(string)
	originalR, err := azure.ParseResourceID(originalPath)
	if err != nil {
		return nil, false, err
	}

	doc, err := f.db.OpenShiftClusters.Get(api.Key(r.resourceID))
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, false, err
	}

	isCreate := doc == nil

	var external api.External
	if isCreate {
		doc = &api.OpenShiftClusterDocument{
			ID: uuid.NewV4().String(),
		}

		external = r.toExternal(&api.OpenShiftCluster{
			ID:   originalPath,
			Name: originalR.ResourceName,
			Type: originalR.ResourceType,
			Properties: api.Properties{
				ProvisioningState: api.ProvisioningStateCreating,
			},
		})

	} else {
		err = f.validateSubscriptionState(doc, api.SubscriptionStateRegistered)
		if err != nil {
			return nil, false, err
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

		doc.OpenShiftCluster.Properties.ProvisioningState = api.ProvisioningStateUpdating

		switch r.method {
		case http.MethodPut:
			external = r.toExternal(&api.OpenShiftCluster{
				ID:   doc.OpenShiftCluster.ID,
				Name: doc.OpenShiftCluster.Name,
				Type: doc.OpenShiftCluster.Type,
				Properties: api.Properties{
					ProvisioningState: doc.OpenShiftCluster.Properties.ProvisioningState,
				},
			})

		case http.MethodPatch:
			external = r.toExternal(doc.OpenShiftCluster)
		}
	}

	err = json.Unmarshal(r.body, &external)
	if err != nil {
		return nil, false, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
	}

	err = external.Validate(r.context, r.resourceID, doc.OpenShiftCluster)
	if err != nil {
		return nil, false, err
	}

	if doc.OpenShiftCluster == nil {
		doc.OpenShiftCluster = &api.OpenShiftCluster{}
	}

	oldID, oldName, oldType := doc.OpenShiftCluster.ID, doc.OpenShiftCluster.Name, doc.OpenShiftCluster.Type
	external.ToInternal(doc.OpenShiftCluster)
	doc.OpenShiftCluster.ID, doc.OpenShiftCluster.Name, doc.OpenShiftCluster.Type = oldID, oldName, oldType

	if isCreate {
		doc.OpenShiftCluster.Key = api.Key(r.resourceID)
		doc.OpenShiftCluster.Properties.Install = &api.Install{}
		// TODO: ResourceGroup should be exposed in external API
		doc.OpenShiftCluster.Properties.ResourceGroup = doc.OpenShiftCluster.Name
		doc.OpenShiftCluster.Properties.DomainName = uuid.NewV4().String()
		doc.OpenShiftCluster.Properties.InfraID = "aro"
		doc.OpenShiftCluster.Properties.SSHKey, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, false, err
		}
		doc.OpenShiftCluster.Properties.StorageSuffix, err = randomLowerCaseAlphanumericString(5)
		if err != nil {
			return nil, false, err
		}

		doc, err = f.db.OpenShiftClusters.Create(doc)
	} else {
		doc, err = f.db.OpenShiftClusters.Update(doc)
	}
	if err != nil {
		return nil, false, err
	}

	doc.OpenShiftCluster.Properties.ServicePrincipalProfile.ClientSecret = ""

	b, err := json.MarshalIndent(r.toExternal(doc.OpenShiftCluster), "", "  ")
	if err != nil {
		return nil, false, err
	}

	return b, isCreate, nil
}

func randomLowerCaseAlphanumericString(n int) (string, error) {
	return randomString("abcdefghijklmnopqrstuvwxyz0123456789", n)
}

func randomString(letterBytes string, n int) (string, error) {
	b := make([]byte, n)
	for i := range b {
		o, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterBytes))))
		if err != nil {
			return "", err
		}
		b[i] = letterBytes[o.Int64()]
	}

	return string(b), nil
}
