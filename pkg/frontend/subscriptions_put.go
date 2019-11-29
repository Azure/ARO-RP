package frontend

import (
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database/cosmosdb"
)

func (f *frontend) putSubscription(w http.ResponseWriter, r *http.Request) {
	log := r.Context().Value(contextKeyLog).(*logrus.Entry)
	vars := mux.Vars(r)

	if r.URL.Query().Get("api-version") != "2.0" {
		api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeInvalidParameter, "", "The resource type 'subscriptions' could not be found for api version '%s'.", r.URL.Query().Get("api-version"))
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
		b, created, err = f._putSubscription(&request{
			context:        r.Context(),
			subscriptionID: vars["subscriptionId"],
			body:           body,
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

func (f *frontend) _putSubscription(r *request) ([]byte, bool, error) {
	doc, err := f.db.Subscriptions.Get(api.Key(r.subscriptionID))
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, false, err
	}

	isCreate := doc == nil

	if isCreate {
		doc = &api.SubscriptionDocument{
			ID:           uuid.NewV4().String(),
			Key:          api.Key(r.subscriptionID),
			Subscription: &api.Subscription{},
		}
	}

	oldState := doc.Subscription.State
	doc.Subscription = &api.Subscription{}

	h := &codec.JsonHandle{
		BasicHandle: codec.BasicHandle{
			DecodeOptions: codec.DecodeOptions{
				ErrorIfNoField: true,
			},
		},
		Indent: 2,
	}

	err = codec.NewDecoderBytes(r.body, h).Decode(&doc.Subscription)
	if err != nil {
		return nil, false, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
	}

	switch doc.Subscription.State {
	case api.SubscriptionStateRegistered, api.SubscriptionStateUnregistered,
		api.SubscriptionStateWarned, api.SubscriptionStateSuspended:
		// allow
	case api.SubscriptionStateDeleted:
		doc.Deleting = true
		doc.Dequeues = 0
	default:
		return nil, false, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidParameter, "state", "The provided state '%s' is invalid.", doc.Subscription.State)
	}

	if oldState == api.SubscriptionStateDeleted && doc.Subscription.State != api.SubscriptionStateDeleted {
		return nil, false, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidSubscriptionState, "", "Request is not allowed in subscription in state '%s'.", oldState)
	}

	if isCreate {
		doc, err = f.db.Subscriptions.Create(doc)
	} else {
		doc, err = f.db.Subscriptions.Update(doc)
	}
	if err != nil {
		return nil, false, err
	}

	var b []byte
	err = codec.NewEncoderBytes(&b, h).Encode(doc.Subscription)
	if err != nil {
		return nil, false, err
	}

	return b, isCreate, nil
}
