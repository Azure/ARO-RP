package frontend

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/metrics"
	"github.com/Azure/ARO-RP/pkg/proxy"
	"github.com/Azure/ARO-RP/pkg/util/bucket"
	"github.com/Azure/ARO-RP/pkg/util/clusterdata"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/heartbeat"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/recover"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

type statusCodeError int

func (err statusCodeError) Error() string {
	return fmt.Sprintf("%d", err)
}

type kubeActionsFactory func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error)

type azureActionsFactory func(*logrus.Entry, env.Interface, *api.OpenShiftCluster, *api.SubscriptionDocument) (adminactions.AzureActions, error)

type ocEnricherFactory func(log *logrus.Entry, dialer proxy.Dialer, m metrics.Emitter) clusterdata.OpenShiftClusterEnricher

type frontend struct {
	auditLog *logrus.Entry
	baseLog  *logrus.Entry
	env      env.Interface

	logMiddleware      middleware.LogMiddleware
	validateMiddleware middleware.ValidateMiddleware
	m                  middleware.MetricsMiddleware
	authMiddleware     middleware.AuthMiddleware

	dbAsyncOperations             database.AsyncOperations
	dbClusterManagerConfiguration database.ClusterManagerConfigurations
	dbOpenShiftClusters           database.OpenShiftClusters
	dbSubscriptions               database.Subscriptions
	dbOpenShiftVersions           database.OpenShiftVersions

	enabledOcpVersions map[string]*api.OpenShiftVersion
	apis               map[string]*api.Version

	lastChangefeed atomic.Value //time.Time
	mu             sync.RWMutex

	aead encryption.AEAD

	kubeActionsFactory  kubeActionsFactory
	azureActionsFactory azureActionsFactory
	ocEnricherFactory   ocEnricherFactory

	quotaValidator QuotaValidator

	l net.Listener
	s *http.Server

	bucketAllocator bucket.Allocator

	startTime time.Time
	ready     atomic.Value

	// these helps us to test and mock easier
	now                          func() time.Time
	systemDataClusterDocEnricher func(*api.OpenShiftClusterDocument, *api.SystemData)

	systemDataSyncSetEnricher              func(*api.ClusterManagerConfigurationDocument, *api.SystemData)
	systemDataMachinePoolEnricher          func(*api.ClusterManagerConfigurationDocument, *api.SystemData)
	systemDataSyncIdentityProviderEnricher func(*api.ClusterManagerConfigurationDocument, *api.SystemData)
	systemDataSecretEnricher               func(*api.ClusterManagerConfigurationDocument, *api.SystemData)

	streamResponder StreamResponder
}

// Runnable represents a runnable object
type Runnable interface {
	Run(context.Context, <-chan struct{}, chan<- struct{})
}

// NewFrontend returns a new runnable frontend
func NewFrontend(ctx context.Context,
	auditLog *logrus.Entry,
	baseLog *logrus.Entry,
	_env env.Interface,
	dbAsyncOperations database.AsyncOperations,
	dbClusterManagerConfiguration database.ClusterManagerConfigurations,
	dbOpenShiftClusters database.OpenShiftClusters,
	dbSubscriptions database.Subscriptions,
	dbOpenShiftVersions database.OpenShiftVersions,
	apis map[string]*api.Version,
	m metrics.Emitter,
	aead encryption.AEAD,
	kubeActionsFactory kubeActionsFactory,
	azureActionsFactory azureActionsFactory,
	ocEnricherFactory ocEnricherFactory) (*frontend, error) {
	f := &frontend{
		logMiddleware: middleware.LogMiddleware{
			EnvironmentName: _env.Environment().Name,
			Location:        _env.Location(),
			Hostname:        _env.Hostname(),
			BaseLog:         baseLog.WithField("component", "access"),
			AuditLog:        auditLog,
		},
		baseLog:  baseLog,
		auditLog: auditLog,
		env:      _env,
		validateMiddleware: middleware.ValidateMiddleware{
			Location: _env.Location(),
			Apis:     api.APIs,
		},
		authMiddleware: middleware.AuthMiddleware{
			AdminAuth: _env.AdminClientAuthorizer(),
			ArmAuth:   _env.ArmClientAuthorizer(),
		},
		dbAsyncOperations:             dbAsyncOperations,
		dbClusterManagerConfiguration: dbClusterManagerConfiguration,
		dbOpenShiftClusters:           dbOpenShiftClusters,
		dbSubscriptions:               dbSubscriptions,
		dbOpenShiftVersions:           dbOpenShiftVersions,
		apis:                          apis,
		m:                             middleware.MetricsMiddleware{Emitter: m},
		aead:                          aead,
		kubeActionsFactory:            kubeActionsFactory,
		azureActionsFactory:           azureActionsFactory,
		ocEnricherFactory:             ocEnricherFactory,
		quotaValidator:                quotaValidator{},

		// add default installation version so it's always supported
		enabledOcpVersions: map[string]*api.OpenShiftVersion{
			version.InstallStream.Version.String(): {
				Properties: api.OpenShiftVersionProperties{
					Version: version.InstallStream.Version.String(),
					Enabled: true,
				},
			},
		},

		bucketAllocator: &bucket.Random{},

		startTime: time.Now(),

		now:                          time.Now,
		systemDataClusterDocEnricher: enrichClusterSystemData,

		systemDataSyncSetEnricher:              enrichSyncSetSystemData,
		systemDataMachinePoolEnricher:          enrichMachinePoolSystemData,
		systemDataSyncIdentityProviderEnricher: enrichSyncIdentityProviderSystemData,
		systemDataSecretEnricher:               enrichSecretSystemData,

		streamResponder: defaultResponder{},
	}

	l, err := f.env.Listen()
	if err != nil {
		return nil, err
	}

	key, certs, err := f.env.ServiceKeyvault().GetCertificateSecret(ctx, env.RPServerSecretName)
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
		SessionTicketsDisabled: true,
		MinVersion:             tls.VersionTLS12,
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
		Queries("api-version", "").
		Subrouter()

	s.Methods(http.MethodDelete).HandlerFunc(f.deleteOpenShiftCluster).Name("deleteOpenShiftCluster")
	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftCluster).Name("getOpenShiftCluster")
	s.Methods(http.MethodPatch).HandlerFunc(f.putOrPatchOpenShiftCluster).Name("putOrPatchOpenShiftCluster")
	s.Methods(http.MethodPut).HandlerFunc(f.putOrPatchOpenShiftCluster).Name("putOrPatchOpenShiftCluster")

	if f.env.FeatureIsSet(env.FeatureEnableOCMEndpoints) {
		s = r.
			Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/{ocmResourceType}/{ocmResourceName}").
			Queries("api-version", "").
			Subrouter()

		s.Methods(http.MethodDelete).HandlerFunc(f.deleteClusterManagerConfiguration).Name("deleteClusterManagerConfiguration")
		s.Methods(http.MethodGet).HandlerFunc(f.getClusterManagerConfiguration).Name("getClusterManagerConfiguration")
		s.Methods(http.MethodPatch).HandlerFunc(f.putOrPatchClusterManagerConfiguration).Name("putOrPatchClusterManagerConfiguration")
		s.Methods(http.MethodPut).HandlerFunc(f.putOrPatchClusterManagerConfiguration).Name("putOrPatchClusterManagerConfiguration")
	}

	s = r.
		Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}").
		Queries("api-version", "").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftClusters).Name("getOpenShiftClusters")

	s = r.
		Path("/subscriptions/{subscriptionId}/providers/{resourceProviderNamespace}/{resourceType}").
		Queries("api-version", "").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getOpenShiftClusters).Name("getOpenShiftClusters")

	s = r.
		Path("/subscriptions/{subscriptionId}/providers/{resourceProviderNamespace}/locations/{location}/operationsstatus/{operationId}").
		Queries("api-version", "").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getAsyncOperationsStatus).Name("getAsyncOperationsStatus")

	s = r.
		Path("/subscriptions/{subscriptionId}/providers/{resourceProviderNamespace}/locations/{location}/operationresults/{operationId}").
		Queries("api-version", "").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getAsyncOperationResult).Name("getAsyncOperationResult")

	s = r.
		Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/listcredentials").
		Queries("api-version", "").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postOpenShiftClusterCredentials).Name("postOpenShiftClusterCredentials")

	s = r.
		Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/listadmincredentials").
		Queries("api-version", "").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postOpenShiftClusterKubeConfigCredentials).Name("postOpenShiftClusterKubeConfigCredentials")

	s = r.
		Path("/subscriptions/{subscriptionId}/providers/{resourceProviderNamespace}/locations/{location}/openshiftversions").
		Queries("api-version", "").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.listInstallVersions).Name("listInstallVersions")

	s = r.
		Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/detectors").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.listAppLensDetectors).Name("listAppLensDetectors")

	s = r.
		Path("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/detectors/{detectorId}").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getAppLensDetector).Name("getAppLensDetector")

	// Admin actions
	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/kubernetesobjects").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getAdminKubernetesObjects).Name("getAdminKubernetesObjects")
	s.Methods(http.MethodPost).HandlerFunc(f.postAdminKubernetesObjects).Name("postAdminKubernetesObjects")
	s.Methods(http.MethodDelete).HandlerFunc(f.deleteAdminKubernetesObjects).Name("deleteAdminKubernetesObjects")

	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/approvecsr").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postAdminOpenShiftClusterApproveCSR).Name("postAdminOpenShiftClusterApproveCSR")

	// Pod logs
	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/kubernetespodlogs").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getAdminKubernetesPodLogs).Name("getAdminKubernetesPodLogs")

	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/resources").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.listAdminOpenShiftClusterResources).Name("listAdminOpenShiftClusterResources")

	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/serialconsole").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getAdminOpenShiftClusterSerialConsole).Name("getAdminOpenShiftClusterSerialConsole")

	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/redeployvm").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postAdminOpenShiftClusterRedeployVM).Name("postAdminOpenShiftClusterRedeployVM")

	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/stopvm").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postAdminOpenShiftClusterStopVM).Name("postAdminOpenShiftClusterStopVM")

	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/startvm").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postAdminOpenShiftClusterStartVM).Name("postAdminOpenShiftClusterStartVM")

	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/upgrade").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postAdminOpenShiftUpgrade).Name("postAdminOpenShiftUpgrade")

	s = r.
		Path("/admin/providers/{resourceProviderNamespace}/{resourceType}").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getAdminOpenShiftClusters).Name("getAdminOpenShiftClusters")

	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/skus").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getAdminOpenShiftClusterVMResizeOptions).Name("getAdminOpenShiftClusterVMResizeOptions")

	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/resize").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postAdminOpenShiftClusterVMResize).Name("postAdminOpenShiftClusterVMResize")

	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/reconcilefailednic").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postAdminReconcileFailedNIC).Name("reconcileFailedNic")

	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/cordonnode").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postAdminOpenShiftClusterCordonNode).Name("postAdminOpenShiftClusterCordonNode")

	s = r.
		Path("/admin/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/drainnode").
		Subrouter()

	s.Methods(http.MethodPost).HandlerFunc(f.postAdminOpenShiftClusterDrainNode).Name("postAdminOpenShiftClusterDrainNode")

	s = r.
		Path("/admin/versions").
		Subrouter()

	s.Methods(http.MethodGet).HandlerFunc(f.getAdminOpenShiftVersions).Name("getAdminOpenShiftVersions")
	s.Methods(http.MethodPut).HandlerFunc(f.putAdminOpenShiftVersion).Name("putAdminOpenShiftVersions")

	// Operations
	s = r.
		Path("/providers/{resourceProviderNamespace}/operations").
		Queries("api-version", "").
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
	r.Use(f.logMiddleware.Log)
	r.Use(f.m.Metrics)
	r.Use(middleware.Panic)
	r.Use(middleware.Headers)
	r.Use(f.validateMiddleware.Validate)
	r.Use(middleware.Body)
	r.Use(middleware.SystemData)

	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeNotFound, "", "The requested path could not be found.")
	})

	r.NotFoundHandler = f.authMiddleware.Authenticate(r.NotFoundHandler)

	unauthenticated := r.NewRoute().Subrouter()
	f.unauthenticatedRoutes(unauthenticated)

	authenticated := r.NewRoute().Subrouter()
	authenticated.Use(f.authMiddleware.Authenticate)

	f.authenticatedRoutes(authenticated)

	return r
}

func (f *frontend) Run(ctx context.Context, stop <-chan struct{}, done chan<- struct{}) {
	defer recover.Panic(f.baseLog)
	go f.changefeed(ctx)

	if stop != nil {
		go func() {
			defer recover.Panic(f.baseLog)

			<-stop

			if !f.env.FeatureIsSet(env.FeatureDisableReadinessDelay) {
				// mark not ready and wait for ((#probes + 1) * interval + longest
				// connection timeout + margin) to stop receiving new connections
				f.baseLog.Print("marking not ready and waiting 80 seconds")
				f.ready.Store(false)
				time.Sleep(80 * time.Second)
			}

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

func adminReply(log *logrus.Entry, w http.ResponseWriter, header http.Header, b []byte, err error) {
	if apiErr, ok := err.(kerrors.APIStatus); ok {
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
		_, _ = w.Write(b)
		_, _ = w.Write([]byte{'\n'})
	}
}

func frontendOperationResultLog(log *logrus.Entry, method string, err error) {
	log = log.WithFields(logrus.Fields{
		"LOGKIND":       "frontendqos",
		"resultType":    utillog.SuccessResultType,
		"operationType": method,
	})

	if err == nil {
		log.Info("front end operation succeeded")
		return
	}

	switch err := err.(type) {
	case *api.CloudError:
		log = log.WithField("resultType", utillog.UserErrorResultType)
	case statusCodeError:
		if int(err) < 300 && int(err) >= 200 {
			log.Info("front end operation succeeded")
			return
		} else if int(err) < 500 {
			log = log.WithField("resultType", utillog.UserErrorResultType)
		} else {
			log = log.WithField("resultType", utillog.ServerErrorResultType)
		}
	default:
		log = log.WithField("resultType", utillog.ServerErrorResultType)
	}

	log = log.WithField("errorDetails", err.Error())
	log.Info("front end operation failed")
}
