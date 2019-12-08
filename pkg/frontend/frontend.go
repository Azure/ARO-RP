package frontend

import (
	"context"
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database"
	"github.com/jim-minter/rp/pkg/env"
	"github.com/jim-minter/rp/pkg/util/recover"
)

const (
	resourceProviderNamespace = "Microsoft.RedHatOpenShift"
	resourceType              = "openShiftClusters"
)

type frontend struct {
	baseLog      *logrus.Entry
	env          env.Interface
	db           *database.Database
	fpAuthorizer autorest.Authorizer

	l net.Listener

	ready atomic.Value
}

// Runnable represents a runnable object
type Runnable interface {
	Run(stop <-chan struct{})
}

// NewFrontend returns a new runnable frontend
func NewFrontend(ctx context.Context, baseLog *logrus.Entry, env env.Interface, db *database.Database) (Runnable, error) {
	var err error

	f := &frontend{
		baseLog: baseLog,
		env:     env,
		db:      db,
	}

	f.fpAuthorizer, err = env.FPAuthorizer(ctx, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

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
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodDelete).HandlerFunc(f.deleteOpenShiftCluster)
	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftCluster)
	s.Methods(http.MethodPatch).HandlerFunc(f.putOrPatchOpenShiftCluster)
	s.Methods(http.MethodPut).HandlerFunc(f.putOrPatchOpenShiftCluster)

	s = r.
		Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}").
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftClusters)

	s = r.
		Path("/subscriptions/{subscriptionId}/providers/{resourceProviderNamespace}/{resourceType}").
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftClusters)

	s = r.
		Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/credentials").
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postOpenShiftClusterCredentials)

	s = r.
		Path("/providers/{resourceProviderNamespace}/operations").
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getOperations)

	s = r.
		Path("/subscriptions/{subscriptionId}").
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodPut).HandlerFunc(f.putSubscription)
}

func (f *frontend) Run(stop <-chan struct{}) {
	defer recover.Panic(f.baseLog)

	go func() {
		defer recover.Panic(f.baseLog)

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

	authenticated.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeNotFound, "", "The requested path could not be found.")
	})

	s := &http.Server{
		Handler:      lowercase(r),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: time.Minute,
		IdleTimeout:  2 * time.Minute,
		ErrorLog:     log.New(f.baseLog.Writer(), "", 0),
	}

	err := s.Serve(f.l)
	f.baseLog.Error(err)
}
