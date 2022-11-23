package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"embed"
	"fmt"
	"strings"
	"time"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/guardrails/config"
	"github.com/Azure/ARO-RP/pkg/util/deployer"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
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
)

const (
	ControllerName               = "GuardRails"
	controllerEnabled            = "aro.guardrails.enabled"
	controllerNamespace          = "aro.guardrails.namespace"
	controllerManaged            = "aro.guardrails.deploy.managed"
	controllerPullSpec           = "aro.guardrails.deploy.pullspec"
	controllerManagerRequestsCPU = "aro.guardrails.deploy.manager.requests.cpu"
	controllerManagerRequestsMem = "aro.guardrails.deploy.manager.requests.mem"
	controllerManagerLimitCPU    = "aro.guardrails.deploy.manager.limit.cpu"
	controllerManagerLimitMem    = "aro.guardrails.deploy.manager.limit.mem"
	controllerAuditRequestsCPU   = "aro.guardrails.deploy.audit.requests.cpu"
	controllerAuditRequestsMem   = "aro.guardrails.deploy.audit.requests.mem"
	controllerAuditLimitCPU      = "aro.guardrails.deploy.audit.limit.cpu"
	controllerAuditLimitMem      = "aro.guardrails.deploy.audit.limit.mem"

	controllerValidatingWebhookFailurePolicy = "aro.guardrails.validatingwebhook.managed"
	controllerValidatingWebhookTimeout       = "aro.guardrails.validatingwebhook.timeoutSeconds"
	controllerMutatingWebhookFailurePolicy   = "aro.guardrails.mutatingwebhook.managed"
	controllerMutatingWebhookTimeout         = "aro.guardrails.mutatingwebhook.timeoutSeconds"

	controllerReconciliationMinutes     = "aro.guardrails.reconciliationMinutes"
	controllerPolicyMachineDenyManaged  = "aro.guardrails.policies.aro-machines-deny.managed"
	controllerPolicyMachineDenyEnforced = "aro.guardrails.policies.aro-machines-deny.enforcement"

	defaultNamespace = "openshift-azure-guardrails"

	defaultManagerRequestsCPU = "100m"
	defaultManagerLimitCPU    = "1000m"
	defaultManagerRequestsMem = "256Mi"
	defaultManagerLimitMem    = "512Mi"
	defaultAuditRequestsCPU   = "100m"
	defaultAuditLimitCPU      = "1000m"
	defaultAuditRequestsMem   = "256Mi"
	defaultAuditLimitMem      = "512Mi"

	defaultReconciliationMinutes = "60"

	defaultValidatingWebhookFailurePolicy = "Ignore"
	defaultValidatingWebhookTimeout       = "3"
	defaultMutatingWebhookFailurePolicy   = "Ignore"
	defaultMutatingWebhookTimeout         = "1"

	gkDeploymentPath  = "staticresources"
	gkTemplatePath    = "gktemplates"
	gkConstraintsPath = "gkconstraints"
)

//go:embed staticresources
var staticFiles embed.FS

//go:embed gktemplates
var gkPolicyTemplates embed.FS

//go:embed gkconstraints
var gkPolicyConraints embed.FS

var pullSecretName = types.NamespacedName{Name: "pull-secret", Namespace: "openshift-config"}

type Reconciler struct {
	arocli           aroclient.Interface
	kubernetescli    kubernetes.Interface
	deployer         deployer.Deployer
	gkPolicyTemplate deployer.Deployer

	readinessPollTime     time.Duration
	readinessTimeout      time.Duration
	dh                    dynamichelper.Interface
	namespace             string
	policyTickerDone      chan bool
	reconciliationMinutes int
}

func NewReconciler(arocli aroclient.Interface, kubernetescli kubernetes.Interface, dh dynamichelper.Interface) *Reconciler {
	return &Reconciler{
		arocli:           arocli,
		kubernetescli:    kubernetescli,
		deployer:         deployer.NewDeployer(kubernetescli, dh, staticFiles, gkDeploymentPath),
		gkPolicyTemplate: deployer.NewDeployer(kubernetescli, dh, gkPolicyTemplates, gkTemplatePath),
		dh:               dh,

		readinessPollTime: 10 * time.Second,
		readinessTimeout:  5 * time.Minute,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	// how to handle the enable/disable sequence of enabled and managed?
	if !instance.Spec.OperatorFlags.GetSimpleBoolean(controllerEnabled) {
		// controller is disabled
		return reconcile.Result{}, nil
	}

	managed := instance.Spec.OperatorFlags.GetWithDefault(controllerManaged, "")

	// If enabled and managed=true, install GuardRails
	// If enabled and managed=false, remove the GuardRails deployment
	// If enabled and managed is missing, do nothing
	if strings.EqualFold(managed, "true") {
		// Deploy the GateKeeper manifests and config
		deployConfig := r.getDefaultDeployConfig(ctx, instance)
		err = r.deployer.CreateOrUpdate(ctx, instance, deployConfig)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Check that GuardRails has become ready, wait up to readinessTimeout (default 5min)
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
	if err := grBuilder.
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{}, predicate.LabelChangedPredicate{})).
		Named(ControllerName).
		Complete(r); err != nil {
		return err
	}
	return nil
}
