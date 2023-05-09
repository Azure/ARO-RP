package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/guardrails/config"
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
}

func NewReconciler(log *logrus.Entry, client client.Client, dh dynamichelper.Interface) *Reconciler {
	return &Reconciler{
		log: log,

		deployer:         deployer.NewDeployer(client, dh, staticFiles, gkDeploymentPath),
		gkPolicyTemplate: deployer.NewDeployer(client, dh, gkPolicyTemplates, gkTemplatePath),
		dh:               dh,

		client: client,

		readinessPollTime: 10 * time.Second,
		readinessTimeout:  5 * time.Minute,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance := &arov1alpha1.Cluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// how to handle the enable/disable sequence of enabled and managed?
	if !instance.Spec.OperatorFlags.GetSimpleBoolean(controllerEnabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running")

	managed := instance.Spec.OperatorFlags.GetWithDefault(controllerManaged, "")

	// If enabled and managed=true, install GuardRails
	// If enabled and managed=false, remove the GuardRails deployment
	// If enabled and managed is missing, do nothing
	if strings.EqualFold(managed, "true") {
		// Check if standard GateKeeper is already running
		// if running, err := r.deployer.IsReady(ctx, "gatekeeper-system", "gatekeeper-audit"); err != nil && running {
		// 	r.log.Warn("standard GateKeeper is running, skipping Guardrails deployment")
		// 	return reconcile.Result{}, nil
		// }

		// Deploy the GateKeeper manifests and config
		deployConfig := r.getDefaultDeployConfig(ctx, instance)
		err = r.deployer.CreateOrUpdate(ctx, instance, deployConfig)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Check Gatekeeper has become ready, wait up to readinessTimeout (default 5min)
		timeoutCtx, cancel := context.WithTimeout(ctx, r.readinessTimeout)
		defer cancel()

		err := wait.PollImmediateUntil(r.readinessPollTime, func() (bool, error) {
			return r.gatekeeperDeploymentIsReady(ctx, deployConfig)
		}, timeoutCtx.Done())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("GateKeeper deployment timed out on Ready: %w", err)
		}
		policyConfig := &config.GuardRailsPolicyConfig{}
		if r.gkPolicyTemplate != nil {
			// Deploy the GateKeeper ConstraintTemplate
			err = r.gkPolicyTemplate.CreateOrUpdate(ctx, instance, policyConfig)
			if err != nil {
				return reconcile.Result{}, err
			}

			err := wait.PollImmediateUntil(r.readinessPollTime, func() (bool, error) {
				return r.gkPolicyTemplate.IsConstraintTemplateReady(ctx, policyConfig)
			}, timeoutCtx.Done())
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("GateKeeper ConstraintTemplates timed out on creation: %w", err)
			}

			// Deploy the GateKeeper Constraint
			err = r.ensurePolicy(ctx, gkPolicyConraints, gkConstraintsPath)
			if err != nil {
				return reconcile.Result{}, err
			}
		}

		// start a ticker to re-enforce gatekeeper policies periodically
		r.startTicker(ctx, instance)
	} else if strings.EqualFold(managed, "false") {
		if r.gkPolicyTemplate != nil {
			// stop the gatekeeper policies re-enforce ticker
			r.stopTicker()

			err = r.removePolicy(ctx, gkPolicyConraints, gkConstraintsPath)
			if err != nil {
				return reconcile.Result{}, err
			}

			err = r.gkPolicyTemplate.Remove(ctx, config.GuardRailsPolicyConfig{})
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		err = r.deployer.Remove(ctx, config.GuardRailsDeploymentConfig{Namespace: r.namespace})
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

// SetupWithManager setup our manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	pullSecretPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return (o.GetName() == pullSecretName.Name && o.GetNamespace() == pullSecretName.Namespace)
	})

	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})

	grBuilder := ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(pullSecretPredicate),
		)

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
