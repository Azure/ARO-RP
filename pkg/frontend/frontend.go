package frontend

import (
	"context"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database"
	"github.com/jim-minter/rp/pkg/env"
)

const (
	resourceProviderNamespace = "Microsoft.RedHatOpenShift"
	resourceType              = "openShiftClusters"
)

type request struct {
	context           context.Context
	method            string
	subscriptionID    string
	resourceID        string
	resourceGroupName string
	resourceName      string
	resourceType      string
	body              []byte
	toExternal        func(*api.OpenShiftCluster) api.External
}

func validateProvisioningState(state api.ProvisioningState, allowedStates ...api.ProvisioningState) error {
	for _, allowedState := range allowedStates {
		if state == allowedState {
			return nil
		}
	}

	return api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeRequestNotAllowed, "", "Request is not allowed in provisioningState '%s'.", state)
}

type frontend struct {
	baseLog *logrus.Entry
	env     env.Interface

	db database.OpenShiftClusters

	l    net.Listener
	tlsl net.Listener

	ready atomic.Value
}

// Runnable represents a runnable object
type Runnable interface {
	Run(stop <-chan struct{})
}

// NewFrontend returns a new runnable frontend
func NewFrontend(ctx context.Context, baseLog *logrus.Entry, env env.Interface, db database.OpenShiftClusters) (Runnable, error) {
	f := &frontend{
		baseLog: baseLog,
		env:     env,
		db:      db,
	}

	var err error
	f.l, err = net.Listen("tcp", ":8080")
	if err != nil {
		return nil, err
	}

	f.tlsl, err = f.env.ListenTLS(ctx)
	if err != nil {
		return nil, err
	}

	f.ready.Store(true)

	return f, nil
}

func (f *frontend) getReady(w http.ResponseWriter, r *http.Request) {
	if f.ready.Load().(bool) && f.env.IsReady() {
		http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
	} else {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// unauthenticatedRouter returns the router which is served via unauthenticated
// HTTP
func (f *frontend) unauthenticatedRouter() *mux.Router {
	r := mux.NewRouter()
	r.Use(f.middleware)

	r.Path("/healthz/ready").Methods(http.MethodGet).HandlerFunc(f.getReady)

	return r
}

// authenticatedRouter returns the router which is served via TLS and protected
// by client certificate authentication
func (f *frontend) authenticatedRouter() *mux.Router {
	r := mux.NewRouter()
	r.Use(f.middleware)

	s := r.
		Path("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}").
		Queries("api-version", "").
		Subrouter()

	s.Methods(http.MethodDelete).HandlerFunc(f.deleteOpenShiftCluster)
	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftCluster)
	s.Methods(http.MethodPatch).HandlerFunc(f.putOrPatchOpenShiftCluster)
	s.Methods(http.MethodPut).HandlerFunc(f.putOrPatchOpenShiftCluster)

	s = r.
		Path("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}").
		Queries("api-version", "").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftClusters)

	s = r.
		Path("/subscriptions/{subscriptionId}/providers/{resourceProviderNamespace}/{resourceType}").
		Queries("api-version", "").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftClusters)

	s = r.
		Path("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/credentials").
		Queries("api-version", "").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postOpenShiftClusterCredentials)

	return r
}

func (f *frontend) Run(stop <-chan struct{}) {
	go func() {
		<-stop
		f.baseLog.Print("marking frontend not ready")
		f.ready.Store(false)
	}()

	go func() {
		err := http.Serve(f.l, f.unauthenticatedRouter())
		f.baseLog.Error(err)
	}()

	err := http.Serve(f.tlsl, f.authenticatedRouter())
	f.baseLog.Error(err)
}
