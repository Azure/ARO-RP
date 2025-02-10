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
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
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
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azsecrets"
	"github.com/Azure/ARO-RP/pkg/util/bucket"
	"github.com/Azure/ARO-RP/pkg/util/clusterdata"
	"github.com/Azure/ARO-RP/pkg/util/encryption"
	"github.com/Azure/ARO-RP/pkg/util/heartbeat"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/log/audit"
	"github.com/Azure/ARO-RP/pkg/util/recover"
)

type statusCodeError int

func (err statusCodeError) Error() string {
	return fmt.Sprintf("%d", err)
}

type frontendDBs interface {
	database.DatabaseGroupWithAsyncOperations
	database.DatabaseGroupWithOpenShiftVersions
	database.DatabaseGroupWithOpenShiftClusters
	database.DatabaseGroupWithAsyncOperations
	database.DatabaseGroupWithSubscriptions
	database.DatabaseGroupWithPlatformWorkloadIdentityRoleSets
	database.DatabaseGroupWithMaintenanceManifests
}

type kubeActionsFactory func(*logrus.Entry, env.Interface, *api.OpenShiftCluster) (adminactions.KubeActions, error)

type azureActionsFactory func(*logrus.Entry, env.Interface, *api.OpenShiftCluster, *api.SubscriptionDocument) (adminactions.AzureActions, error)
type appLensActionsFactory func(*logrus.Entry, env.Interface, *api.OpenShiftCluster, *api.SubscriptionDocument) (adminactions.AppLensActions, error)

type frontend struct {
	auditLog *logrus.Entry
	baseLog  *logrus.Entry
	env      env.Interface

	logMiddleware         middleware.LogMiddleware
	validateMiddleware    middleware.ValidateMiddleware
	m                     middleware.MetricsMiddleware
	authMiddleware        middleware.AuthMiddleware
	apiVersionMiddleware  middleware.ApiVersionValidator
	maintenanceMiddleware middleware.MaintenanceMiddleware

	dbGroup frontendDBs

	defaultOcpVersion                         string // always enabled
	enabledOcpVersions                        map[string]*api.OpenShiftVersion
	availablePlatformWorkloadIdentityRoleSets map[string]*api.PlatformWorkloadIdentityRoleSet
	apis                                      map[string]*api.Version

	lastOcpVersionsChangefeed                      atomic.Value //time.Time
	lastPlatformWorkloadIdentityRoleSetsChangefeed atomic.Value
	ocpVersionsMu                                  sync.RWMutex
	platformWorkloadIdentityRoleSetsMu             sync.RWMutex

	aead encryption.AEAD

	hiveClusterManager    hive.ClusterManager
	hiveSyncSetManager    hive.SyncSetManager
	kubeActionsFactory    kubeActionsFactory
	azureActionsFactory   azureActionsFactory
	appLensActionsFactory appLensActionsFactory

	skuValidator       SkuValidator
	quotaValidator     QuotaValidator
	providersValidator ProvidersValidator

	clusterEnricher clusterdata.BestEffortEnricher

	l net.Listener
	s *http.Server

	bucketAllocator bucket.Allocator

	startTime time.Time
	ready     atomic.Value

	// these helps us to test and mock easier
	now                          func() time.Time
	systemDataClusterDocEnricher func(*api.OpenShiftClusterDocument, *api.SystemData)

	streamResponder StreamResponder
}

// Runnable represents a runnable object
type Runnable interface {
	Run(context.Context, <-chan struct{}, chan<- struct{})
}

// TODO: Get the number of function parameters under control :D
// NewFrontend returns a new runnable frontend
func NewFrontend(ctx context.Context,
	auditLog *logrus.Entry,
	baseLog *logrus.Entry,
	outelAuditClient audit.Client,
	_env env.Interface,
	dbGroup frontendDBs,
	apis map[string]*api.Version,
	m metrics.Emitter,
	clusterm metrics.Emitter,
	aead encryption.AEAD,
	hiveClusterManager hive.ClusterManager,
	hiveSyncSetManager hive.SyncSetManager,
	kubeActionsFactory kubeActionsFactory,
	azureActionsFactory azureActionsFactory,
	appLensActionsFactory appLensActionsFactory,
	enricher clusterdata.BestEffortEnricher,
) (*frontend, error) {
	f := &frontend{
		logMiddleware: middleware.LogMiddleware{
			EnvironmentName:  _env.Environment().Name,
			Location:         _env.Location(),
			Hostname:         _env.Hostname(),
			BaseLog:          baseLog.WithField("component", "access"),
			AuditLog:         auditLog,
			OutelAuditClient: outelAuditClient,
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
			Log:         baseLog,
			EnableMISE:  _env.FeatureIsSet(env.FeatureEnableMISE),
			EnforceMISE: _env.FeatureIsSet(env.FeatureEnforceMISE),
			AdminAuth:   _env.AdminClientAuthorizer(),
			ArmAuth:     _env.ArmClientAuthorizer(),
			MiseAuth:    _env.MISEAuthorizer(),
		},
		dbGroup:               dbGroup,
		apis:                  apis,
		m:                     middleware.MetricsMiddleware{Emitter: m},
		maintenanceMiddleware: middleware.MaintenanceMiddleware{Emitter: clusterm},
		aead:                  aead,
		hiveClusterManager:    hiveClusterManager,
		hiveSyncSetManager:    hiveSyncSetManager,
		kubeActionsFactory:    kubeActionsFactory,
		azureActionsFactory:   azureActionsFactory,
		appLensActionsFactory: appLensActionsFactory,

		quotaValidator:     quotaValidator{},
		skuValidator:       skuValidator{},
		providersValidator: providersValidator{},

		clusterEnricher: enricher,

		enabledOcpVersions:                        map[string]*api.OpenShiftVersion{},
		availablePlatformWorkloadIdentityRoleSets: map[string]*api.PlatformWorkloadIdentityRoleSet{},

		bucketAllocator: &bucket.Random{},

		startTime: time.Now(),

		now:                          time.Now,
		systemDataClusterDocEnricher: enrichClusterSystemData,

		streamResponder: defaultResponder{},
	}

	l, err := f.env.Listen()
	if err != nil {
		return nil, err
	}

	certificate, err := f.env.ServiceKeyvault().GetSecret(ctx, env.RPServerSecretName, "", nil)
	if err != nil {
		return nil, err
	}

	key, certs, err := azsecrets.ParseSecretAsCertificate(certificate)
	if err != nil {
		return nil, err
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{
			{
				PrivateKey: key,
			},
		},
		NextProtos:             []string{"h2", "http/1.1"},
		ClientAuth:             tls.RequestClientCert,
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
					r.Delete("/", f.deleteOpenShiftCluster)
					r.Get("/", f.getOpenShiftCluster)

					if f.env.IsLocalDevelopmentMode() {
						r.With(middleware.MockMSIMiddleware).Patch("/", f.putOrPatchOpenShiftCluster)
						r.With(middleware.MockMSIMiddleware).Put("/", f.putOrPatchOpenShiftCluster)
					} else {
						r.Patch("/", f.putOrPatchOpenShiftCluster)
						r.Put("/", f.putOrPatchOpenShiftCluster)
					}

					r.Post("/listcredentials", f.postOpenShiftClusterCredentials)

					r.Post("/listadmincredentials", f.postOpenShiftClusterKubeConfigCredentials)
				})

				r.Get("/detectors", f.listAppLensDetectors)

				r.Get("/detectors/{detectorId}", f.getAppLensDetector)
			})
		})

		r.Route("/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/deployments/{deploymentName}/preflight", func(r chi.Router) {
			r.Use(f.apiVersionMiddleware.ValidatePreflightAPIVersion)
			r.Post("/", f.preflightValidation)
		})

		r.Route("/providers/{resourceProviderNamespace}", func(r chi.Router) {
			r.Use(f.apiVersionMiddleware.ValidateAPIVersion)

			r.Get("/{resourceType}", f.getOpenShiftClusters)

			r.Route("/locations/{location}", func(r chi.Router) {
				r.Get("/operationsstatus/{operationId}", f.getAsyncOperationsStatus)

				r.Get("/operationresults/{operationId}", f.getAsyncOperationResult)

				r.Get("/openshiftversions", f.listInstallVersions)
				r.Get("/openshiftversions/{openshiftVersion}", f.getInstallVersion)

				r.Get("/platformworkloadidentityrolesets", f.listPlatformWorkloadIdentityRoleSets)
				r.Get("/platformworkloadidentityrolesets/{openShiftMinorVersion}", f.getPlatformWorkloadIdentityRoleSet)
			})
		})
	})

	//Admin Actions

	r.Route("/admin", func(r chi.Router) {
		r.Route("/versions", func(r chi.Router) {
			r.Get("/", f.getAdminOpenShiftVersions)
			r.Put("/", f.putAdminOpenShiftVersion)
		})
		r.Route("/platformworkloadidentityrolesets", func(r chi.Router) {
			r.Get("/", f.getAdminPlatformWorkloadIdentityRoleSets)
			r.Put("/", f.putAdminPlatformWorkloadIdentityRoleSet)
		})
		r.Get("/supportedvmsizes", f.supportedvmsizes)

		r.Route("/maintenancemanifests", func(r chi.Router) {
			r.Get("/queued", f.getAdminQueuedMaintManifests)
		})
		r.Route("/hivesyncset", func(r chi.Router) {
			r.Get("/", f.listAdminHiveSyncSet)
			r.Get("/syncsetname/{syncsetname}", f.getAdminHiveSyncSet)
		})

		r.Route("/subscriptions/{subscriptionId}", func(r chi.Router) {
			r.Route("/resourcegroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}", func(r chi.Router) {
				// Etcd recovery
				r.With(f.maintenanceMiddleware.UnplannedMaintenanceSignal).Post("/etcdrecovery", f.postAdminOpenShiftClusterEtcdRecovery)

				// Kubernetes objects
				r.Get("/kubernetesobjects", f.getAdminKubernetesObjects)
				r.With(f.maintenanceMiddleware.UnplannedMaintenanceSignal).Post("/kubernetesobjects", f.postAdminKubernetesObjects)
				r.With(f.maintenanceMiddleware.UnplannedMaintenanceSignal).Delete("/kubernetesobjects", f.deleteAdminKubernetesObjects)

				r.With(f.maintenanceMiddleware.UnplannedMaintenanceSignal).Post("/approvecsr", f.postAdminOpenShiftClusterApproveCSR)

				// Pod logs
				r.Get("/kubernetespodlogs", f.getAdminKubernetesPodLogs)

				r.Get("/resources", f.listAdminOpenShiftClusterResources)

				r.Get("/serialconsole", f.getAdminOpenShiftClusterSerialConsole)

				r.Get("/clusterdeployment", f.getAdminHiveClusterDeployment)

				r.Get("/clustersync", f.getAdminHiveClusterSync)

				r.With(f.maintenanceMiddleware.UnplannedMaintenanceSignal).Post("/redeployvm", f.postAdminOpenShiftClusterRedeployVM)

				r.With(f.maintenanceMiddleware.UnplannedMaintenanceSignal).Post("/stopvm", f.postAdminOpenShiftClusterStopVM)

				r.With(f.maintenanceMiddleware.UnplannedMaintenanceSignal).Post("/startvm", f.postAdminOpenShiftClusterStartVM)

				r.Get("/skus", f.getAdminOpenShiftClusterVMResizeOptions)

				// We don't emit unplanned maintenance signal for resize since it is only used for planned maintenance
				r.Post("/resize", f.postAdminOpenShiftClusterVMResize)

				r.With(f.maintenanceMiddleware.UnplannedMaintenanceSignal).Post("/reconcilefailednic", f.postAdminReconcileFailedNIC)

				r.With(f.maintenanceMiddleware.UnplannedMaintenanceSignal).Post("/cordonnode", f.postAdminOpenShiftClusterCordonNode)

				r.With(f.maintenanceMiddleware.UnplannedMaintenanceSignal).Post("/drainnode", f.postAdminOpenShiftClusterDrainNode)

				r.With(f.maintenanceMiddleware.UnplannedMaintenanceSignal).Post("/etcdcertificaterenew", f.postAdminOpenShiftClusterEtcdCertificateRenew)
				r.With(f.maintenanceMiddleware.UnplannedMaintenanceSignal).Post("/deletemanagedresource", f.postAdminOpenShiftDeleteManagedResource)

				// MIMO
				r.Route("/maintenancemanifests", func(r chi.Router) {
					r.Get("/", f.getAdminMaintManifests)
					r.Put("/", f.putAdminMaintManifestCreate)
					r.Route("/{manifestId}", func(r chi.Router) {
						r.Get("/", f.getSingleAdminMaintManifest)
						r.Delete("/", f.deleteAdminMaintManifest)
						r.Post("/cancel", f.postAdminMaintManifestCancel)
					})
				})
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
	go f.changefeedOcpVersions(ctx)
	go f.changefeedRoleSets(ctx)

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

	var statusCode int

	switch err := err.(type) {
	case *api.CloudError:
		statusCode = err.StatusCode
	case statusCodeError:
		statusCode = int(err)
	default:
		statusCode = 500
	}

	resultType := utillog.MapStatusCodeToResultType(statusCode)
	log = log.WithField("resultType", resultType)

	if resultType == utillog.SuccessResultType {
		log.Info("front end operation succeeded")
		return
	}

	log = log.WithField("errorDetails", err.Error())
	log.Info("front end operation failed")
}

// resourceIdFromURLParams returns an Azure Resource ID built out of the
// individual parameters of the URL.
func resourceIdFromURLParams(r *http.Request) string {
	subID, resType, resProvider, resName, resGroupName := chi.URLParam(r, "subscriptionId"),
		chi.URLParam(r, "resourceType"),
		chi.URLParam(r, "resourceProviderNamespace"),
		chi.URLParam(r, "resourceName"),
		chi.URLParam(r, "resourceGroupName")

	return strings.ToLower(azure.Resource{
		SubscriptionID: subID,
		ResourceGroup:  resGroupName,
		ResourceType:   resType,
		ResourceName:   resName,
		Provider:       resProvider,
	}.String())
}
