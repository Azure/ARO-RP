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
	"sigs.k8s.io/controller-runtime/pkg/source"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/muo/config"
	"github.com/Azure/ARO-RP/pkg/util/deployer"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/pullsecret"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

const (
	ControllerName = "ManagedUpgradeOperator"

	controllerEnabled                = "rh.srep.muo.enabled"
	controllerManaged                = "rh.srep.muo.managed"
	controllerPullSpec               = "rh.srep.muo.deploy.pullspec"
	controllerForceLocalOnly         = "rh.srep.muo.deploy.forceLocalOnly"
	controllerOcmBaseURL             = "rh.srep.muo.deploy.ocmBaseUrl"
	controllerOcmBaseURLDefaultValue = "https://api.openshift.com"

	pullSecretOCMKey = "cloud.openshift.com"
)

//go:embed staticresources
var staticFiles embed.FS

var pullSecretName = types.NamespacedName{Name: "pull-secret", Namespace: "openshift-config"}

type MUODeploymentConfig struct {
	Pullspec     string
	ConnectToOCM bool
	OCMBaseURL   string
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

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(controllerEnabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running")

	managed := instance.Spec.OperatorFlags.GetWithDefault(controllerManaged, "")

	// If enabled and managed=true, install MUO
	// If enabled and managed=false, remove the MUO deployment
	// If enabled and managed is missing, do nothing
	if strings.EqualFold(managed, "true") {
		// apply the default pullspec if the flag is empty or missing
		pullSpec := instance.Spec.OperatorFlags.GetWithDefault(controllerPullSpec, "")
		if pullSpec == "" {
			pullSpec = version.MUOImage(instance.Spec.ACRDomain)
		}

		config := &config.MUODeploymentConfig{
			Pullspec: pullSpec,
		}

		disableOCM := instance.Spec.OperatorFlags.GetSimpleBoolean(controllerForceLocalOnly)
		if !disableOCM {
			useOCM := func() bool {
				userSecret := &corev1.Secret{}
				err = r.client.Get(ctx, pullSecretName, userSecret)
				if err != nil {
					// if a pullsecret doesn't exist/etc, fallback to local
					return false
				}

				parsedKeys, err := pullsecret.UnmarshalSecretData(userSecret)
				if err != nil {
					// if we can't parse the pullsecret, fallback to local
					return false
				}

				// check for the key that connects the cluster to OCM (since
				// clusters may have a RH registry pull secret but not the OCM
				// one if they choose)
				_, foundKey := parsedKeys[pullSecretOCMKey]
				return foundKey
			}()

			// if we have a valid pullsecret, enable connected MUO
			if useOCM {
				config.EnableConnected = true
				config.OCMBaseURL = instance.Spec.OperatorFlags.GetWithDefault(controllerOcmBaseURL, controllerOcmBaseURLDefaultValue)
			}
		}

		// Deploy the MUO manifests and config
		err = r.deployer.CreateOrUpdate(ctx, instance, config)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Check that MUO has become ready, wait up to readinessTimeout (default 5min)
		timeoutCtx, cancel := context.WithTimeout(ctx, r.readinessTimeout)
		defer cancel()

		err := wait.PollImmediateUntil(r.readinessPollTime, func() (bool, error) {
			return r.deployer.IsReady(ctx, "openshift-managed-upgrade-operator", "managed-upgrade-operator")
		}, timeoutCtx.Done())
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
	pullSecretPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return (o.GetName() == pullSecretName.Name && o.GetNamespace() == pullSecretName.Namespace)
	})

	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})

	muoBuilder := ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(pullSecretPredicate),
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
