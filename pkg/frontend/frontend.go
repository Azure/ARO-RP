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

type frontend struct {
	baseLog *logrus.Entry
	env     env.Interface

	db *database.Database

	l net.Listener

	ready atomic.Value
}

// Runnable represents a runnable object
type Runnable interface {
	Run(stop <-chan struct{})
}

// NewFrontend returns a new runnable frontend
func NewFrontend(ctx context.Context, baseLog *logrus.Entry, env env.Interface, db *database.Database) (Runnable, error) {
	f := &frontend{
		baseLog: baseLog,
		env:     env,
		db:      db,
	}

	var err error
	f.l, err = f.env.ListenTLS(ctx)
	if err != nil {
		return nil, err
	}

	f.ready.Store(true)

	return f, nil
}

func (f *frontend) getReady(w http.ResponseWriter, r *http.Request) {
	if f.ready.Load().(bool) && f.env.IsReady() {
		api.WriteCloudError(w, &api.CloudError{StatusCode: http.StatusOK})
	} else {
		api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
	}
}

func (f *frontend) unauthenticatedRoutes(r *mux.Router) {
	r.Path("/healthz/ready").Methods(http.MethodGet).HandlerFunc(f.getReady)
}

func (f *frontend) authenticatedRoutes(r *mux.Router) {
	s := r.
		Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}").
		Queries("api-version", "").
		Subrouter()

	s.Methods(http.MethodDelete).HandlerFunc(f.deleteOpenShiftCluster)
	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftCluster)
	s.Methods(http.MethodPatch).HandlerFunc(f.putOrPatchOpenShiftCluster)
	s.Methods(http.MethodPut).HandlerFunc(f.putOrPatchOpenShiftCluster)

	s = r.
		Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}").
		Queries("api-version", "").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftClusters)

	s = r.
		Path("/subscriptions/{subscriptionId}/providers/{resourceProviderNamespace}/{resourceType}").
		Queries("api-version", "").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftClusters)

	s = r.
		Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/credentials").
		Queries("api-version", "").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postOpenShiftClusterCredentials)

	s = r.
		Path("/providers/{resourceProviderNamespace}/operations").
		Queries("api-version", "").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getOperations)

	s = r.
		Path("/subscriptions/{subscriptionId}").
		Queries("api-version", "").
		Subrouter()

	s.Methods(http.MethodPut).HandlerFunc(f.putSubscription)
}

func (f *frontend) Run(stop <-chan struct{}) {
	go func() {
		<-stop
		f.baseLog.Print("marking frontend not ready")
		f.ready.Store(false)
	}()

	r := mux.NewRouter()
	r.Use(f.middleware)

	unauthenticated := r.NewRoute().Subrouter()
	f.unauthenticatedRoutes(unauthenticated)

	authenticated := r.NewRoute().Subrouter()
	authenticated.Use(f.env.Authenticated)
	f.authenticatedRoutes(authenticated)

	err := http.Serve(f.l, lowercase(r))
	f.baseLog.Error(err)
}
