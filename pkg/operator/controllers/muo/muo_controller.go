package muo

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"embed"
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

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/muo/config"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
	"github.com/Azure/ARO-RP/pkg/util/deployer"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	ControllerName     = "ManagedUpgradeOperator"
	controllerPullSpec = "rh.srep.muo.deploy.pullspec"
)

//go:embed staticresources
var staticFiles embed.FS

type MUODeploymentConfig struct {
	Pullspec string
}

type Reconciler struct {
	log *logrus.Entry

	deployer deployer.Deployer

	client client.Client

	readinessPollTime time.Duration
	readinessTimeout  time.Duration
}

func NewReconciler(log *logrus.Entry, client client.Client, dh dynamichelper.Interface) *Reconciler {
	return &Reconciler{
		log: log,

		deployer: deployer.NewDeployer(client, dh, staticFiles, "staticresources"),

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

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.MuoEnabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running")

	managed := instance.Spec.OperatorFlags.GetWithDefault(operator.MuoManaged, "")

	// If enabled and managed=true, install MUO
	// If enabled and managed=false, remove the MUO deployment
	// If enabled and managed is missing, do nothing
	if strings.EqualFold(managed, "true") {
		// apply the default pullspec if the flag is empty or missing
		pullSpec := instance.Spec.OperatorFlags.GetWithDefault(controllerPullSpec, "")
		if pullSpec == "" {
			pullSpec = version.MUOImage(instance.Spec.ACRDomain)
		}

		usePodSecurityAdmission, err := operator.ShouldUsePodSecurityStandard(ctx, r.client)
		if err != nil {
			return reconcile.Result{}, err
		}

		config := &config.MUODeploymentConfig{
			SupportsPodSecurityAdmission: usePodSecurityAdmission,
			Pullspec:                     pullSpec,
		}

		// Deploy the MUO manifests and config
		err = r.deployer.CreateOrUpdate(ctx, instance, config)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Check that MUO has become ready, wait up to readinessTimeout (default 5min)
		timeoutCtx, cancel := context.WithTimeout(ctx, r.readinessTimeout)
		defer cancel()

		err = wait.PollUntilContextCancel(timeoutCtx, r.readinessPollTime, true, func(ctx context.Context) (bool, error) {
			return r.deployer.IsReady(ctx, "openshift-managed-upgrade-operator", "managed-upgrade-operator")
		})
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("managed Upgrade Operator deployment timed out on Ready: %w", err)
		}
	} else if strings.EqualFold(managed, "false") {
		err := r.deployer.Remove(ctx, config.MUODeploymentConfig{})
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

// SetupWithManager setup our manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	muoBuilder := ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		Watches(
			&corev1.Secret{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicates.PullSecret),
		)

	resources, err := r.deployer.Template(&config.MUODeploymentConfig{}, staticFiles)
	if err != nil {
		return err
	}

	for _, i := range resources {
		o, ok := i.(client.Object)
		if ok {
			muoBuilder.Owns(o)
		}
	}

	return muoBuilder.
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{}, predicate.LabelChangedPredicate{})).
		Named(ControllerName).
		Complete(r)
}
