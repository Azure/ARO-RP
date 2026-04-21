package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/guardrails/config"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
	"github.com/Azure/ARO-RP/pkg/util/deployer"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

type Reconciler struct {
	log              *logrus.Entry
	deployer         deployer.Deployer
	gkPolicyTemplate deployer.Deployer
	client           client.Client

	readinessPollTime     time.Duration
	readinessTimeout      time.Duration
	dh                    dynamichelper.Interface
	namespace             string
	gkTickerDone          chan struct{}
	vapTickerDone         chan struct{}
	tickerMu              sync.Mutex
	reconciliationMinutes int
	cleanupNeeded         bool
	kubernetescli         kubernetes.Interface
}

func NewReconciler(log *logrus.Entry, client client.Client, dh dynamichelper.Interface, k8scli kubernetes.Interface) *Reconciler {
	return &Reconciler{
		log: log,

		deployer:         deployer.NewDeployer(client, dh, staticFiles, gkDeploymentPath),
		gkPolicyTemplate: deployer.NewDeployer(client, dh, gkPolicyTemplates, gkTemplatePath),
		dh:               dh,

		client: client,

		readinessPollTime: 10 * time.Second,
		readinessTimeout:  5 * time.Minute,
		cleanupNeeded:     false,
		kubernetescli:     k8scli,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance := &arov1alpha1.Cluster{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance); err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.GuardrailsEnabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running")

	managed := instance.Spec.OperatorFlags.GetWithDefault(operator.GuardrailsDeployManaged, "")

	lt417, err := r.VersionLT417(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	// managed=false: clean up whatever policy mechanism this version uses
	if strings.EqualFold(managed, "false") {
		return r.cleanupManaged(ctx, instance, lt417)
	}

	// managed is blank/missing: no action
	if !strings.EqualFold(managed, "true") {
		r.log.Warnf("unrecognised managed flag (%s), doing nothing", managed)
		return reconcile.Result{}, nil
	}

	// Pre-4.17 clusters use the Gatekeeper / Rego workflow
	if lt417 {
		return r.deployGatekeeper(ctx, instance)
	}

	// v4.17+ — migrate away from Gatekeeper if it is still running
	if r.gatekeeperCleanupNeeded(ctx, instance) {
		if err := r.cleanupGatekeeper(ctx, instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Deploy VAP policies according to per-policy feature flags
	if err := r.deployVAP(ctx); err != nil {
		return reconcile.Result{}, err
	}

	r.startVAPTicker(ctx, instance)

	return reconcile.Result{}, nil
}

// deployGatekeeper handles the managed=true path for clusters < v4.17.
func (r *Reconciler) deployGatekeeper(ctx context.Context, instance *arov1alpha1.Cluster) (ctrl.Result, error) {
	if ns, err := r.getGatekeeperDeployedNs(ctx, instance); err == nil && ns != "" {
		r.log.Warnf("Found another GateKeeper deployed in ns %s, aborting Guardrails", ns)
		return reconcile.Result{}, nil
	}

	deployConfig := r.getDefaultDeployConfig(ctx, instance)
	if err := r.deployer.CreateOrUpdate(ctx, instance, deployConfig); err != nil {
		return reconcile.Result{}, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, r.readinessTimeout)
	defer cancel()

	if err := wait.PollImmediateUntil(r.readinessPollTime, func() (bool, error) {
		return r.gatekeeperDeploymentIsReady(ctx, deployConfig)
	}, timeoutCtx.Done()); err != nil {
		return reconcile.Result{}, fmt.Errorf("GateKeeper deployment timed out on Ready: %w", err)
	}

	r.cleanupNeeded = true

	if r.gkPolicyTemplate != nil {
		policyConfig := &config.GuardRailsPolicyConfig{}

		if err := r.gkPolicyTemplate.CreateOrUpdate(ctx, instance, policyConfig); err != nil {
			return reconcile.Result{}, err
		}

		if err := wait.PollImmediateUntil(r.readinessPollTime, func() (bool, error) {
			return r.gkPolicyTemplate.IsConstraintTemplateReady(ctx, policyConfig)
		}, timeoutCtx.Done()); err != nil {
			return reconcile.Result{}, fmt.Errorf("GateKeeper ConstraintTemplates timed out on creation: %w", err)
		}

		if err := r.ensurePolicy(ctx, gkPolicyConstraints, gkConstraintsPath); err != nil {
			return reconcile.Result{}, err
		}
	}

	r.startGKTicker(ctx, instance)
	return reconcile.Result{}, nil
}

// cleanupManaged handles the managed=false path. The resources to remove
// depend on the cluster version: pre-4.17 uses Gatekeeper, 4.17+ uses VAP
// (and may also need leftover Gatekeeper resources removed).
func (r *Reconciler) cleanupManaged(ctx context.Context, instance *arov1alpha1.Cluster, lt417 bool) (ctrl.Result, error) {
	if lt417 {
		return r.cleanupGatekeeperManaged(ctx, instance)
	}

	// v4.17+: remove VAP policies
	r.stopVAPTicker()
	if err := r.removeAllVAP(ctx); err != nil {
		r.log.Warnf("failed to remove VAP policies: %s", err.Error())
	}

	// also clean up Gatekeeper if it is still present (upgrade scenario)
	if r.gatekeeperCleanupNeeded(ctx, instance) {
		if err := r.cleanupGatekeeper(ctx, instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

// cleanupGatekeeperManaged is the managed=false cleanup for pre-4.17 clusters.
// It preserves the safety check that avoids destroying a customer-deployed
// Gatekeeper in a different namespace, then delegates to cleanupGatekeeper.
func (r *Reconciler) cleanupGatekeeperManaged(ctx context.Context, instance *arov1alpha1.Cluster) (ctrl.Result, error) {
	if !r.cleanupNeeded {
		if ns, err := r.getGatekeeperDeployedNs(ctx, instance); err == nil && ns != "" {
			r.log.Warn("Skipping cleanup as it is not safe and may destroy customer's gatekeeper resources")
			return reconcile.Result{}, nil
		}
	}

	if err := r.cleanupGatekeeper(ctx, instance); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

// SetupWithManager setup our manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	grBuilder := ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{})))

	resources, err := r.deployer.Template(&config.GuardRailsDeploymentConfig{}, staticFiles)
	if err != nil {
		return err
	}

	for _, i := range resources {
		o, ok := i.(client.Object)
		if ok {
			grBuilder.Owns(o)
		}
	}

	// we won't listen for changes on policies, since we only want to reconcile on a timer anyway
	return grBuilder.
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{}, predicate.LabelChangedPredicate{})).
		Named(ControllerName).
		Complete(r)
}
