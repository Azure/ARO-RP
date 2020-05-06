package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/gorilla/mux"
	"github.com/jmespath/go-jmespath"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/kubeactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/compute"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/features"
	"github.com/Azure/ARO-RP/pkg/util/bucket"
	"github.com/Azure/ARO-RP/pkg/util/clusterdata"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/heartbeat"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

type statusCodeError int

func (err statusCodeError) Error() string {
	return fmt.Sprintf("%d", err)
}

type kubeActionsFactory func(*logrus.Entry, env.Interface) kubeactions.Interface
type resourcesClientFactory func(subscriptionID string, authorizer autorest.Authorizer) features.ResourcesClient
type computeClientFactory func(subscriptionID string, authorizer autorest.Authorizer) compute.VirtualMachinesClient

type frontend struct {
	baseLog *logrus.Entry
	env     env.Interface
	db      *database.Database
	apis    map[string]*api.Version
	m       metrics.Interface
	cipher  encryption.Cipher

	ocEnricher             clusterdata.OpenShiftClusterEnricher
	kubeActionsFactory     kubeActionsFactory
	resourcesClientFactory resourcesClientFactory
	computeClientFactory   computeClientFactory

	l net.Listener
	s *http.Server

	bucketAllocator bucket.Allocator

	startTime time.Time
	ready     atomic.Value
}

// Runnable represents a runnable object
type Runnable interface {
	Run(context.Context, <-chan struct{}, chan<- struct{})
}

// NewFrontend returns a new runnable frontend
func NewFrontend(ctx context.Context, baseLog *logrus.Entry, _env env.Interface, db *database.Database, apis map[string]*api.Version, m metrics.Interface, cipher encryption.Cipher, kubeActionsFactory kubeActionsFactory, resourcesClientFactory resourcesClientFactory, computeClientFactory computeClientFactory) (Runnable, error) {
	f := &frontend{
		baseLog:                baseLog,
		env:                    _env,
		db:                     db,
		apis:                   apis,
		m:                      m,
		cipher:                 cipher,
		kubeActionsFactory:     kubeActionsFactory,
		resourcesClientFactory: resourcesClientFactory,
		computeClientFactory:   computeClientFactory,

		ocEnricher: clusterdata.NewBestEffortEnricher(baseLog, _env, m),

		bucketAllocator: &bucket.Random{},

		startTime: time.Now(),
	}

	l, err := f.env.Listen()
	if err != nil {
		return nil, err
	}

	key, certs, err := f.env.GetCertificateSecret(ctx, env.RPServerSecretName)
	if err != nil {
		return nil, err
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{
			{
				PrivateKey: key,
			},
		},
		NextProtos: []string{"h2", "http/1.1"},
		ClientAuth: tls.RequestClientCert,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		},
		PreferServerCipherSuites: true,
		SessionTicketsDisabled:   true,
		MinVersion:               tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
	}

	for _, cert := range certs {
		config.Certificates[0].Certificate = append(config.Certificates[0].Certificate, cert.Raw)
	}

	f.l = tls.NewListener(l, config)

	f.ready.Store(true)
	return f, nil
}

func (f *frontend) unauthenticatedRoutes(r *mux.Router) {
	r.Path("/healthz/ready").Methods(http.MethodGet).HandlerFunc(f.getReady).Name("getReady")
}

func (f *frontend) authenticatedRoutes(r *mux.Router) {
	s := r.
		Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}").
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodDelete).HandlerFunc(f.deleteOpenShiftCluster).Name("deleteOpenShiftCluster")
	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftCluster).Name("getOpenShiftCluster")
	s.Methods(http.MethodPatch).HandlerFunc(f.putOrPatchOpenShiftCluster).Name("putOrPatchOpenShiftCluster")
	s.Methods(http.MethodPut).HandlerFunc(f.putOrPatchOpenShiftCluster).Name("putOrPatchOpenShiftCluster")

	s = r.
		Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}").
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftClusters).Name("getOpenShiftClusters")

	s = r.
		Path("/subscriptions/{subscriptionId}/providers/{resourceProviderNamespace}/{resourceType}").
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftClusters).Name("getOpenShiftClusters")

	s = r.
		Path("/subscriptions/{subscriptionId}/providers/{resourceProviderNamespace}/locations/{location}/operationsstatus/{operationId}").
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getAsyncOperationsStatus).Name("getAsyncOperationsStatus")

	s = r.
		Path("/subscriptions/{subscriptionId}/providers/{resourceProviderNamespace}/locations/{location}/operationresults/{operationId}").
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getAsyncOperationResult).Name("getAsyncOperationResult")

	s = r.
		Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/listcredentials").
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postOpenShiftClusterCredentials).Name("postOpenShiftClusterCredentials")

	// Admin actions
	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/kubernetesobjects").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getAdminKubernetesObjects).Name("getAdminKubernetesObjects")
	s.Methods(http.MethodPost).HandlerFunc(f.postAdminKubernetesObjects).Name("postAdminKubernetesObjects")
	s.Methods(http.MethodDelete).HandlerFunc(f.deleteAdminKubernetesObjects).Name("deleteAdminKubernetesObjects")

	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/upgrade").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postAdminOpenShiftClusterUpgrade).Name("postAdminOpenShiftClusterUpgrade")

	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/resources").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.listAdminOpenShiftClusterResources).Name("listAdminOpenShiftClusterResources")

	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/mustgather").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postAdminOpenShiftClusterMustGather).Name("postAdminOpenShiftClusterMustGather")

	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/restartvm").
		Subrouter() // remove after Geneva actions have been updated

	s.Methods(http.MethodPost).HandlerFunc(f.postAdminOpenShiftClusterRedeployVM).Name("postAdminOpenShiftClusterRestartVM") // remove after Geneva actions have been updated

	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/redeployvm").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postAdminOpenShiftClusterRedeployVM).Name("postAdminOpenShiftClusterRedeployVM")

	s = r.
		Path("/admin/providers/{resourceProviderNamespace}/{resourceType}").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getAdminOpenShiftClusters).Name("getAdminOpenShiftClusters")

	// Operations
	s = r.
		Path("/providers/{resourceProviderNamespace}/operations").
		Queries("api-version", "{api-version}").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getOperations).Name("getOperations")

	s = r.
		Path("/subscriptions/{subscriptionId}").
		Queries("api-version", "2.0").
		Subrouter()

	s.Methods(http.MethodPut).HandlerFunc(f.putSubscription).Name("putSubscription")
}

func (f *frontend) setupRouter() *mux.Router {
	r := mux.NewRouter()
	r.Use(middleware.Log(f.baseLog.WithField("component", "access")))
	r.Use(middleware.Metrics(f.m))
	r.Use(middleware.Panic)
	r.Use(middleware.Headers(f.env))
	r.Use(middleware.Validate(f.env, f.apis))
	r.Use(middleware.Body)

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeNotFound, "", "The requested path could not be found.")
	})
	r.NotFoundHandler = middleware.Authenticated(f.env)(r.NotFoundHandler)

	unauthenticated := r.NewRoute().Subrouter()
	f.unauthenticatedRoutes(unauthenticated)

	authenticated := r.NewRoute().Subrouter()
	authenticated.Use(middleware.Authenticated(f.env))
	f.authenticatedRoutes(authenticated)

	return r
}

func (f *frontend) Run(ctx context.Context, stop <-chan struct{}, done chan<- struct{}) {
	defer recover.Panic(f.baseLog)

	if stop != nil {
		go func() {
			defer recover.Panic(f.baseLog)

			<-stop

			// mark not ready and wait for ((#probes + 1) * interval + longest
			// connection timeout + margin) to stop receiving new connections
			f.baseLog.Print("marking not ready and waiting 80 seconds")
			f.ready.Store(false)
			time.Sleep(80 * time.Second)

			f.baseLog.Print("exiting")
			close(done)
		}()
	}

	f.s = &http.Server{
		Handler:     middleware.Lowercase(f.setupRouter()),
		ReadTimeout: 10 * time.Second,
		IdleTimeout: 2 * time.Minute,
		ErrorLog:    log.New(f.baseLog.Writer(), "", 0),
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	go heartbeat.EmitHeartbeat(f.baseLog, f.m, "frontend.heartbeat", stop, f.checkReady)

	err := f.s.Serve(f.l)
	if err != http.ErrServerClosed {
		f.baseLog.Error(err)
	}
}

func adminJmespathFilter(result []byte, jpath *jmespath.JMESPath) ([]byte, error) {
	if jpath == nil {
		return result, nil
	}

	var o interface{}
	err := json.Unmarshal(result, &o)
	if err != nil {
		return nil, err
	}

	filtered, err := jpath.Search(o)
	if err != nil {
		return nil, api.NewCloudError(http.StatusBadRequest, api.CloudErrorCodeJMESPathSearchFailed, "", "The JMESPath search returned an error: '%s'", err)
	}

	return json.Marshal(filtered)
}

func adminReply(log *logrus.Entry, w http.ResponseWriter, header http.Header, b []byte, err error) {
	if apiErr, ok := err.(errors.APIStatus); ok {
		status := apiErr.Status()

		var target string
		if status.Details != nil {
			gk := schema.GroupKind{
				Group: status.Details.Group,
				Kind:  status.Details.Kind,
			}

			target = fmt.Sprintf("%s/%s", gk, status.Details.Name)
		}

		err = &api.CloudError{
			StatusCode: int(status.Code),
			CloudErrorBody: &api.CloudErrorBody{
				Code:    string(status.Reason),
				Message: status.Message,
				Target:  target,
			},
		}
	}

	reply(log, w, header, b, err)
}

func reply(log *logrus.Entry, w http.ResponseWriter, header http.Header, b []byte, err error) {
	for k, v := range header {
		w.Header()[k] = v
	}

	if err != nil {
		switch err := err.(type) {
		case *api.CloudError:
			log.Info(err)
			api.WriteCloudError(w, err)
			return
		case statusCodeError:
			w.WriteHeader(int(err))
		default:
			log.Error(err)
			api.WriteError(w, http.StatusInternalServerError, api.CloudErrorCodeInternalServerError, "", "Internal server error.")
			return
		}
	}

	if b != nil {
		w.Write(b)
		w.Write([]byte{'\n'})
	}
}
