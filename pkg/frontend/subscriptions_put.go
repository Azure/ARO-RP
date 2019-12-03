package frontend

import (
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

	if vars["api-version"] != "2.0" {
		api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeInvalidParameter, "", "The resource type 'subscriptions' could not be found for api version '%s'.", vars["api-version"])
		return
	}

	var err error
	r, err = readBody(w, r)
	if err != nil {
		api.WriteCloudError(w, err.(*api.CloudError))
		return
	}

	var b []byte
	var created bool
	err = cosmosdb.RetryOnPreconditionFailed(func() error {
		b, created, err = f._putSubscription(r)
		return err
	})
	if err == nil && created {
		w.WriteHeader(http.StatusCreated)
	}

	reply(log, w, b, err)
}

func (f *frontend) _putSubscription(r *http.Request) ([]byte, bool, error) {
	body := r.Context().Value(contextKeyBody).([]byte)
	vars := mux.Vars(r)

	doc, err := f.db.Subscriptions.Get(api.Key(vars["subscriptionId"]))
	if err != nil && !cosmosdb.IsErrorStatusCode(err, http.StatusNotFound) {
		return nil, false, err
	}

	isCreate := doc == nil

	if isCreate {
		doc = &api.SubscriptionDocument{
			ID:           uuid.NewV4().String(),
			Key:          api.Key(vars["subscriptionId"]),
			Subscription: &api.Subscription{},
		}
	}

	oldState := doc.Subscription.State

	h := &codec.JsonHandle{
		BasicHandle: codec.BasicHandle{
			DecodeOptions: codec.DecodeOptions{
				ErrorIfNoField: true,
			},
		},
		Indent: 4,
	}

	doc.Subscription = &api.Subscription{}
	err = codec.NewDecoderBytes(body, h).Decode(&doc.Subscription)
	if err != nil {
		return nil, false, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeInvalidRequestContent, "", "The request content was invalid and could not be deserialized: %q.", err)
	}

	switch doc.Subscription.State {
	case api.SubscriptionStateRegistered, api.SubscriptionStateUnregistered,
		api.SubscriptionStateWarned, api.SubscriptionStateSuspended:
		// allow
	case api.SubscriptionStateDeleted:
		doc.Deleting = true
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
