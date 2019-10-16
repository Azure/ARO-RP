package frontend

import (
	"net"
	"net/http"
	"sync/atomic"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/jim-minter/rp/pkg/api"
	"github.com/jim-minter/rp/pkg/database"
)

const (
	resourceProviderNamespace = "RedHat.OpenShift"
	resourceType              = "OpenShiftClusters"
)

type request struct {
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

	db   database.OpenShiftClusters
	apis map[api.APIVersionType]func(*api.OpenShiftCluster) api.External

	l net.Listener

	healthy atomic.Value
}

// Runnable represents a runnable object
type Runnable interface {
	Run(stop <-chan struct{})
}

// NewFrontend returns a new runnable frontend
func NewFrontend(baseLog *logrus.Entry, l net.Listener, db database.OpenShiftClusters, apis map[api.APIVersionType]func(*api.OpenShiftCluster) api.External) Runnable {
	f := &frontend{
		baseLog: baseLog,
		db:      db,
		apis:    apis,

		l: l,
	}

	f.healthy.Store(true)

	return f
}

func (f *frontend) health(w http.ResponseWriter, r *http.Request) {
	if f.healthy.Load().(bool) {
		http.Error(w, "", http.StatusOK)
	} else {
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func (f *frontend) Run(stop <-chan struct{}) {
	r := mux.NewRouter()
	r.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
	r.Path("/health").Methods(http.MethodGet).HandlerFunc(f.health)

	s := r.
		Path("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}").
		Queries("api-version", "").
		Subrouter()

	s.Use(f.middleware)
	s.Methods(http.MethodDelete).HandlerFunc(f.deleteOpenShiftCluster)
	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftCluster)
	s.Methods(http.MethodPatch).HandlerFunc(f.putOrPatchOpenShiftCluster)
	s.Methods(http.MethodPut).HandlerFunc(f.putOrPatchOpenShiftCluster)

	s = r.
		Path("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}").
		Queries("api-version", "").
		Subrouter()

	s.Use(f.middleware)
	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftClusters)

	s = r.
		Path("/subscriptions/{subscriptionId}/providers/{resourceProviderNamespace}/{resourceType}").
		Queries("api-version", "").
		Subrouter()

	s.Use(f.middleware)
	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftClusters)

	s = r.
		Path("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/credentials").
		Queries("api-version", "").
		Subrouter()

	s.Use(f.middleware)
	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftClusterCredentials)

	go func() {
		<-stop
		f.baseLog.Println("marking frontend unhealthy")
		f.healthy.Store(false)
	}()

	err := http.Serve(f.l, r)
	f.baseLog.Error(err)
}
