package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"embed"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/guardrails/config"
	"github.com/Azure/ARO-RP/pkg/util/deployer"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/version"
	"github.com/sirupsen/logrus"
)

const (
	ControllerName      = "GuardRails"
	controllerEnabled   = "aro.guardrails.enabled"        // boolean, false by default
	controllerNamespace = "aro.guardrails.namespace"      // string
	controllerManaged   = "aro.guardrails.deploy.managed" // trinary, do-nothing by default
	controllerPullSpec  = "aro.guardrails.deploy.pullspec"
	// controllerRequestCPU            = "aro.guardrails.deploy.requests.cpu"
	// controllerRequestMem            = "aro.guardrails.deploy.requests.mem"
	// controllerLimitCPU              = "aro.guardrails.deploy.limits.cpu"
	// controllerLimitMem              = "aro.guardrails.deploy.limits.mem"
	// controllerWebhookManaged        = "aro.guardrails.webhook.managed"        // trinary, do-nothing by default
	// controllerWebhookTimeout        = "aro.guardrails.webhook.timeoutSeconds" // int, 3 by default (as per upstream)
	// controllerReconciliationMinutes = "aro.guardrails.reconciliationMinutes"  // int, 60 by default.

	defaultNamespace = "openshift-azure-guardrails"
)

//go:embed staticresources
var staticFiles embed.FS

var pullSecretName = types.NamespacedName{Name: "pull-secret", Namespace: "openshift-config"}

type Reconciler struct {
	arocli        aroclient.Interface
	kubernetescli kubernetes.Interface
	deployer      deployer.Deployer

	readinessPollTime time.Duration
	readinessTimeout  time.Duration
}

func NewReconciler(arocli aroclient.Interface, kubernetescli kubernetes.Interface, dh dynamichelper.Interface) *Reconciler {
	return &Reconciler{
		arocli:        arocli,
		kubernetescli: kubernetescli,
		deployer:      deployer.NewDeployer(kubernetescli, dh, staticFiles, "staticresources"),

		readinessPollTime: 10 * time.Second,
		readinessTimeout:  5 * time.Minute,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logrus.Printf("\x1b[%dm guardrails::Reconcile enter 0\x1b[0m", 31)
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		logrus.Printf("\x1b[%dm guardrails:: reconcile error getting %s\x1b[0m", 31, err.Error())
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(controllerEnabled) {
		// controller is disabled
		return reconcile.Result{}, nil
	}

	managed := instance.Spec.OperatorFlags.GetWithDefault(controllerManaged, "")

	// If enabled and managed=true, install GuardRails
	// If enabled and managed=false, remove the GuardRails deployment
	// If enabled and managed is missing, do nothing
	if strings.EqualFold(managed, "true") {
		// apply the default pullspec if the flag is empty or missing
		pullSpec := instance.Spec.OperatorFlags.GetWithDefault(controllerPullSpec, "")
		if pullSpec == "" {
			pullSpec = version.GateKeeperImage(instance.Spec.ACRDomain)
		}
		// apply the default namespace if the flag is empty or missing
		namespace := instance.Spec.OperatorFlags.GetWithDefault(controllerNamespace, defaultNamespace)

		config := &config.GuardRailsDeploymentConfig{
			Pullspec:  pullSpec,
			Namespace: namespace,
		}

		// Deploy the GateKeeper manifests and config
		err = r.deployer.CreateOrUpdate(ctx, instance, config)
		if err != nil {
			logrus.Printf("\x1b[%dm guardrails:: reconcile error updating %s\x1b[0m", 31, err.Error())
			return reconcile.Result{}, err
		}

		// Check that GuardRails has become ready, wait up to readinessTimeout (default 5min)
		timeoutCtx, cancel := context.WithTimeout(ctx, r.readinessTimeout)
		defer cancel()

		err := wait.PollImmediateUntil(r.readinessPollTime, func() (bool, error) {
			if ready, err := r.deployer.IsReady(ctx, config.Namespace, "gatekeeper-audit"); !ready || err != nil {
				return ready, err
			}
			return r.deployer.IsReady(ctx, config.Namespace, "gatekeeper-controller-manager")
		}, timeoutCtx.Done())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("GateKeeper deployment timed out on Ready: %w", err)
		}
	} else if strings.EqualFold(managed, "false") {
		err := r.deployer.Remove(ctx, config.GuardRailsDeploymentConfig{Namespace: instance.Spec.OperatorFlags.GetWithDefault(controllerNamespace, defaultNamespace)})
		if err != nil {
			logrus.Printf("\x1b[%dm guardrails:: reconcile error removing %s\x1b[0m", 31, err.Error())
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

	return grBuilder.
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{}, predicate.LabelChangedPredicate{})).
		Named(ControllerName).
		Complete(r)
}
