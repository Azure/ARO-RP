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

	"github.com/go-chi/chi/v5"
	chiMiddlewares "github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/database"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/frontend/adminactions"
	"github.com/Azure/ARO-RP/pkg/frontend/middleware"
	"github.com/Azure/ARO-RP/pkg/hive"
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

	logMiddleware        middleware.LogMiddleware
	validateMiddleware   middleware.ValidateMiddleware
	m                    middleware.MetricsMiddleware
	authMiddleware       middleware.AuthMiddleware
	apiVersionMiddleware middleware.ApiVersionValidator

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

	hiveClusterManager  hive.ClusterManager
	kubeActionsFactory  kubeActionsFactory
	azureActionsFactory azureActionsFactory
	ocEnricherFactory   ocEnricherFactory

	skuValidator       SkuValidator
	quotaValidator     QuotaValidator
	providersValidator ProvidersValidator

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
	hiveClusterManager hive.ClusterManager,
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
		apiVersionMiddleware: middleware.ApiVersionValidator{
			APIs: api.APIs,
		},
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
		hiveClusterManager:            hiveClusterManager,
		kubeActionsFactory:            kubeActionsFactory,
		azureActionsFactory:           azureActionsFactory,
		ocEnricherFactory:             ocEnricherFactory,
		quotaValidator:                quotaValidator{},
		skuValidator:                  skuValidator{},
		providersValidator:            providersValidator{},

		// add default installation version so it's always supported
		enabledOcpVersions: map[string]*api.OpenShiftVersion{
			version.DefaultInstallStream.Version.String(): {
				Properties: api.OpenShiftVersionProperties{
					Version: version.DefaultInstallStream.Version.String(),
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

func (f *frontend) chiUnauthenticatedRoutes(router chi.Router) {
	router.Get("/healthz/ready", f.getReady)
}

func (f *frontend) chiAuthenticatedRoutes(router chi.Router) {
	r := router.With(f.authMiddleware.Authenticate)

	r.Route("/subscriptions/{subscriptionId}", func(r chi.Router) {
		r.Route("/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}", func(r chi.Router) {
			r.With(f.apiVersionMiddleware.ValidateAPIVersion).Get("/", f.getOpenShiftClusters)

			r.Route("/{resourceName}", func(r chi.Router) {
				r.With(f.apiVersionMiddleware.ValidateAPIVersion).Route("/", func(r chi.Router) {
					// With API version check
					if f.env.FeatureIsSet(env.FeatureEnableOCMEndpoints) {
						r.Route("/{ocmResourceType}",
							func(r chi.Router) {
								r.Delete("/{ocmResourceName}", f.deleteClusterManagerConfiguration)
								r.Get("/{ocmResourceName}", f.getClusterManagerConfiguration)
								r.Patch("/{ocmResourceName}", f.putOrPatchClusterManagerConfiguration)
								r.Put("/{ocmResourceName}", f.putOrPatchClusterManagerConfiguration)
							},
						)
					}

					r.Delete("/", f.deleteOpenShiftCluster)
					r.Get("/", f.getOpenShiftCluster)
					r.Patch("/", f.putOrPatchOpenShiftCluster)
					r.Put("/", f.putOrPatchOpenShiftCluster)

					r.Post("/listcredentials", f.postOpenShiftClusterCredentials)

					r.Post("/listadmincredentials", f.postOpenShiftClusterKubeConfigCredentials)
				})

				r.Get("/detectors", f.listAppLensDetectors)

				r.Get("/detectors/{detectorId}", f.getAppLensDetector)
			})
		})

		r.Route("/providers/{resourceProviderNamespace}", func(r chi.Router) {
			r.Use(f.apiVersionMiddleware.ValidateAPIVersion)

			r.Get("/{resourceType}", f.getOpenShiftClusters)

			r.Route("/locations/{location}", func(r chi.Router) {
				r.Get("/operationsstatus/{operationId}", f.getAsyncOperationsStatus)

				r.Get("/operationresults/{operationId}", f.getAsyncOperationResult)

				r.Get("/openshiftversions", f.listInstallVersions)
			})
		})
	})

	//Admin Actions

	r.Route("/admin", func(r chi.Router) {
		r.Route("/versions", func(r chi.Router) {
			r.Get("/", f.getAdminOpenShiftVersions)
			r.Put("/", f.putAdminOpenShiftVersion)
		})
		r.Get("/supportedvmsizes", f.supportedvmsizes)

		r.Route("/subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/kubernetesobjects",
			func(r chi.Router) {
				r.Get("/", f.getAdminKubernetesObjects)
				r.Post("/", f.postAdminKubernetesObjects)
				r.Delete("/", f.deleteAdminKubernetesObjects)
			},
		)

		r.Route("/subscriptions/{subscriptionId}", func(r chi.Router) {
			r.Route("/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}", func(r chi.Router) {
				r.Post("/approvecsr", f.postAdminOpenShiftClusterApproveCSR)

				// Pod logs
				r.Get("/kubernetespodlogs", f.getAdminKubernetesPodLogs)

				r.Get("/resources", f.listAdminOpenShiftClusterResources)

				r.Get("/serialconsole", f.getAdminOpenShiftClusterSerialConsole)

				r.Get("/clusterdeployment", f.getAdminHiveClusterDeployment)

				r.Post("/redeployvm", f.postAdminOpenShiftClusterRedeployVM)

				r.Post("/stopvm", f.postAdminOpenShiftClusterStopVM)

				r.Post("/startvm", f.postAdminOpenShiftClusterStartVM)

				r.Post("/upgrade", f.postAdminOpenShiftUpgrade)

				r.Get("/skus", f.getAdminOpenShiftClusterVMResizeOptions)

				r.Post("/resize", f.postAdminOpenShiftClusterVMResize)

				r.Post("/reconcilefailednic", f.postAdminReconcileFailedNIC)

				r.Post("/cordonnode", f.postAdminOpenShiftClusterCordonNode)

				r.Post("/drainnode", f.postAdminOpenShiftClusterDrainNode)
			})
		})

		// Operations
		r.Route("/providers/{resourceProviderNamespace}", func(r chi.Router) {
			r.Get("/{resourceType}", f.getAdminOpenShiftClusters)
		})
	})

	r.Put("/subscriptions/{subscriptionId}", f.putSubscription)

	r.With(f.apiVersionMiddleware.ValidateAPIVersion).Get("/providers/{resourceProviderNamespace}/operations", f.getOperations)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	api.WriteError(w, http.StatusNotFound, api.CloudErrorCodeNotFound, "", "The requested path could not be found.")
}

func (f *frontend) setupRouter() chi.Router {
	chiRouter := chi.NewMux()

	chiRouter.Use(chiMiddlewares.CleanPath)

	chiRouter.NotFound(f.authMiddleware.Authenticate(http.HandlerFunc(notFound)).ServeHTTP)
	registered := chiRouter.With(
		chiMiddlewares.CleanPath,
		f.logMiddleware.Log,
		f.m.Metrics,
		middleware.Panic,
		middleware.Headers,
		f.validateMiddleware.Validate,
		middleware.Body,
		middleware.SystemData)
	f.chiAuthenticatedRoutes(registered)
	f.chiUnauthenticatedRoutes(registered)

	return chiRouter
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
