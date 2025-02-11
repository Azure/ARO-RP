package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
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
	policyTickerDone      chan bool
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
	err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// how to handle the enable/disable sequence of enabled and managed?
	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.GuardrailsEnabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running")

	managed := instance.Spec.OperatorFlags.GetWithDefault(operator.GuardrailsDeployManaged, "")

	// If enabled and managed=true, install GuardRails
	// If enabled and managed=false, remove the GuardRails deployment
	// If enabled and managed is missing, do nothing
	if strings.EqualFold(managed, "true") {
		if ns, err := r.getGatekeeperDeployedNs(ctx, instance); err == nil && ns != "" {
			r.log.Warnf("Found another GateKeeper deployed in ns %s, aborting Guardrails", ns)
			return reconcile.Result{}, nil
		}

		// Deploy the GateKeeper manifests and config
		deployConfig := r.getDefaultDeployConfig(ctx, instance)
		err = r.deployer.CreateOrUpdate(ctx, instance, deployConfig)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Check Gatekeeper has become ready, wait up to readinessTimeout (default 5min)
		timeoutCtx, cancel := context.WithTimeout(ctx, r.readinessTimeout)
		defer cancel()

		err := wait.PollUntilContextCancel(timeoutCtx, r.readinessPollTime, true, func(ctx context.Context) (bool, error) {
			return r.gatekeeperDeploymentIsReady(ctx, deployConfig)
		})
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("GateKeeper deployment timed out on Ready: %w", err)
		}
		r.cleanupNeeded = true
		policyConfig := &config.GuardRailsPolicyConfig{}
		if r.gkPolicyTemplate != nil {
			// Deploy the GateKeeper ConstraintTemplate
			err = r.gkPolicyTemplate.CreateOrUpdate(ctx, instance, policyConfig)
			if err != nil {
				return reconcile.Result{}, err
			}

			err := wait.PollUntilContextCancel(timeoutCtx, r.readinessPollTime, true, func(ctx context.Context) (bool, error) {
				return r.gkPolicyTemplate.IsConstraintTemplateReady(ctx, policyConfig)
			})
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("GateKeeper ConstraintTemplates timed out on creation: %w", err)
			}

			// Deploy the GateKeeper Constraint
			err = r.ensurePolicy(ctx, gkPolicyConstraints, gkConstraintsPath)
			if err != nil {
				return reconcile.Result{}, err
			}
		}

		// start a ticker to re-enforce gatekeeper policies periodically
		r.startTicker(ctx, instance)
	} else if strings.EqualFold(managed, "false") {
		if !r.cleanupNeeded {
			if ns, err := r.getGatekeeperDeployedNs(ctx, instance); err == nil && ns != "" {
				// resources were *not* created by guardrails, plus another gatekeeper deployed
				//
				// guardrails didn't get deployed most likely due to another gatekeeper is deployed by customer
				// this is to avoid the accidental deletion of gatekeeper CRDs that were deployed by customer
				// the unnamespaced gatekeeper CRDs were possibly created by a customised gatekeeper, hence cannot ramdomly delete them.
				r.log.Warn("Skipping cleanup as it is not safe and may destroy customer's gatekeeper resources")
				return reconcile.Result{}, nil
			}
		}

		if r.gkPolicyTemplate != nil {
			// stop the gatekeeper policies re-enforce ticker
			r.stopTicker()

			err = r.removePolicy(ctx, gkPolicyConstraints, gkConstraintsPath)
			if err != nil {
				r.log.Warnf("failed to remove Constraints with error %s", err.Error())
			}

			err = r.gkPolicyTemplate.Remove(ctx, config.GuardRailsPolicyConfig{})
			if err != nil {
				r.log.Warnf("failed to remove ConstraintTemplates with error %s", err.Error())
			}
		}
		err = r.deployer.Remove(ctx, config.GuardRailsDeploymentConfig{Namespace: r.namespace})
		if err != nil {
			r.log.Warnf("failed to remove deployer with error %s", err.Error())
			return reconcile.Result{}, err
		}
		r.cleanupNeeded = false
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
