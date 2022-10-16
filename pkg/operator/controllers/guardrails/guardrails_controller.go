package guardrails

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
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
	templatePath     = "gkpolicies/templates"
	constraintspath  = "gkpolicies/constraints"
)

//go:embed staticresources
var staticFiles embed.FS

//go:embed gkpolicies
var policyFiles embed.FS

var pullSecretName = types.NamespacedName{Name: "pull-secret", Namespace: "openshift-config"}

type Reconciler struct {
	arocli        aroclient.Interface
	kubernetescli kubernetes.Interface
	deployer      deployer.Deployer
	gkPolicy      deployer.Deployer

	readinessPollTime time.Duration
	readinessTimeout  time.Duration
	//log               logr.Logger
	restConfig *rest.Config
	// used to invoke dynamichelper.NewGVRResolver()
	logentry *logrus.Entry
}

func NewReconciler(arocli aroclient.Interface, kubernetescli kubernetes.Interface, dh dynamichelper.Interface) *Reconciler {
	return &Reconciler{
		arocli:        arocli,
		kubernetescli: kubernetescli,
		deployer:      deployer.NewDeployer(kubernetescli, dh, staticFiles, "staticresources"),
		gkPolicy:      deployer.NewDeployer(kubernetescli, dh, policyFiles, "gkpolicies"),

		readinessPollTime: 10 * time.Second,
		readinessTimeout:  5 * time.Minute,
		logentry:          utillog.GetLogger(), // anyway to get a logrus entry?
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
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

		deployConfig := &config.GuardRailsDeploymentConfig{
			Pullspec:  pullSpec,
			Namespace: namespace,
		}

		// Deploy the GateKeeper manifests and config
		err = r.deployer.CreateOrUpdate(ctx, instance, deployConfig)
		if err != nil {
			logrus.Printf("\x1b[%dm guardrails:: reconcile error updating %s\x1b[0m", 31, err.Error())
			return reconcile.Result{}, err
		}

		// Check that GuardRails has become ready, wait up to readinessTimeout (default 5min)
		timeoutCtx, cancel := context.WithTimeout(ctx, r.readinessTimeout)
		defer cancel()

		err := wait.PollImmediateUntil(r.readinessPollTime, func() (bool, error) {
			if ready, err := r.deployer.IsReady(ctx, deployConfig.Namespace, "gatekeeper-audit"); !ready || err != nil {
				return ready, err
			}
			return r.deployer.IsReady(ctx, deployConfig.Namespace, "gatekeeper-controller-manager")
		}, timeoutCtx.Done())
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("GateKeeper deployment timed out on Ready: %w", err)
		}

		// TODO: check if can move setup/remove logic to deployer
		// policyConfig := &config.GuardRailsPolicyConfig{}

		// Deploy the GateKeeper policies
		// err = r.gkPolicy.CreateOrUpdate(ctx, instance, policyConfig)
		// if err != nil {
		// 	return reconcile.Result{}, err
		// }

		err = r.setupPolicy(templatePath)
		if err != nil {
			logrus.Printf("\x1b[%dm guardrails:: reconcile error setup template %s\x1b[0m", 31, err.Error())
			return reconcile.Result{}, err
		}

		err = r.setupPolicy(constraintspath)
		if err != nil {
			logrus.Printf("\x1b[%dm guardrails:: reconcile error setup constraints %s\x1b[0m", 31, err.Error())
			return reconcile.Result{}, err
		}

		// // Check that GuardRails has become ready, wait up to readinessTimeout (default 5min)
		// timeoutCtx, cancel = context.WithTimeout(ctx, r.readinessTimeout)
		// defer cancel()

		// err = wait.PollImmediateUntil(r.readinessPollTime, func() (bool, error) {
		// 	// TODO: fix policy checks
		// 	if ready, err := r.gkPolicy.IsReady(ctx, policyConfig.Namespace, "gk-policy-1"); !ready || err != nil {
		// 		return ready, err
		// 	}
		// 	return r.gkPolicy.IsReady(ctx, policyConfig.Namespace, "gk-policy-2")
		// }, timeoutCtx.Done())
		// if err != nil {
		// 	return reconcile.Result{}, fmt.Errorf("GateKeeper policy timed out on Ready: %w", err)
		// }

	} else if strings.EqualFold(managed, "false") {
		// TODO: check if can move setup/remove logic to deployer
		// err := r.gkPolicy.Remove(ctx, config.GuardRailsPolicyConfig{})
		// if err != nil {
		// 	return reconcile.Result{}, err
		// }
		err = r.removePolicy(constraintspath)
		if err != nil {
			logrus.Printf("\x1b[%dm guardrails:: reconcile error removing constraints %s\x1b[0m", 31, err.Error())
			return reconcile.Result{}, err
		}

		err = r.removePolicy(templatePath)
		if err != nil {
			logrus.Printf("\x1b[%dm guardrails:: reconcile error removing template %s\x1b[0m", 31, err.Error())
			return reconcile.Result{}, err
		}

		err = r.deployer.Remove(ctx, config.GuardRailsDeploymentConfig{Namespace: instance.Spec.OperatorFlags.GetWithDefault(controllerNamespace, defaultNamespace)})
		if err != nil {
			logrus.Printf("\x1b[%dm guardrails:: reconcile error removing deployment %s\x1b[0m", 31, err.Error())
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

// SetupWithManager setup our manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {

	r.restConfig = mgr.GetConfig()
	// r.log = mgr.GetLogger()
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
		logrus.Printf("\x1b[%dm guardrails::SetupWithManager deployment failed %v 0\x1b[0m", 31, err)
		return err
	}
	return nil
}

func (r *Reconciler) setupPolicy(path string) error {
	ctx := context.Background()
	template, err := template.ParseFS(policyFiles, filepath.Join(path, "*"))
	if err != nil {
		return err
	}

	buffer := new(bytes.Buffer)
	for _, templ := range template.Templates() {
		err := templ.Execute(buffer, nil)
		if err != nil {
			return err
		}
		//logrus.Printf("\x1b[%dm buffer %v \x1b[0m", 31, buffer.String())
		// data, err := io.ReadAll(buffer)
		// if err != nil {
		// 	return err
		// }
		data := buffer.Bytes()
		logrus.Printf("\x1b[%dm guardrails:: setting up template %v: %s \x1b[0m", 31, templ, string(data))
		obj := &unstructured.Unstructured{}
		json, err := yaml.YAMLToJSON(data)
		if err != nil {
			return err
		}
		err = obj.UnmarshalJSON(json)
		if err != nil {
			return err
		}
		logrus.Println("Unmarshal result: \n", obj)

		// TODO fix logrus.Entry
		// is it ok to use 	log := utillog.GetLogger()  ???
		// as it seems no way to convert logr.logger to logrus.Entry?
		gvrResolver, err := dynamichelper.NewGVRResolver(r.logentry, r.restConfig)
		if err != nil {
			return err
		}

		dyn, err := dynamic.NewForConfig(r.restConfig)
		if err != nil {
			return err
		}

		gvr, err := gvrResolver.Resolve(obj.GroupVersionKind().GroupKind().String(), obj.GroupVersionKind().Version)
		if err != nil {
			return err
		}

		// is update needed here?
		// _, err = dyn.Resource(*gvr).Namespace(obj.GetNamespace()).Update(ctx, obj, metav1.UpdateOptions{})
		// if !kerrors.IsNotFound(err) {
		// 	return err
		// }
		if _, err = dyn.Resource(*gvr).Namespace(obj.GetNamespace()).Create(ctx, obj, metav1.CreateOptions{}); err != nil && !strings.Contains(err.Error(), "already exists") {
			return err
		}
	}
	return nil
}

func (r *Reconciler) removePolicy(path string) error {
	ctx := context.Background()
	template, err := template.ParseFS(policyFiles, filepath.Join(path, "*"))
	if err != nil {
		return err
	}

	buffer := new(bytes.Buffer)
	for _, templ := range template.Templates() {
		err := templ.Execute(buffer, nil)
		if err != nil {
			return err
		}
		data := buffer.Bytes()
		logrus.Printf("\x1b[%dm guardrails:: removing template %v: %s \x1b[0m", 31, templ, string(data))
		obj := &unstructured.Unstructured{}
		json, err := yaml.YAMLToJSON(data)
		if err != nil {
			return err
		}
		err = obj.UnmarshalJSON(json)
		if err != nil {
			return err
		}
		logrus.Println("Unmarshal result: \n", obj)

		// TODO fix logrus.Entry
		gvrResolver, err := dynamichelper.NewGVRResolver(r.logentry, r.restConfig)
		if err != nil {
			return err
		}

		dyn, err := dynamic.NewForConfig(r.restConfig)
		if err != nil {
			return err
		}

		gvr, err := gvrResolver.Resolve(obj.GroupVersionKind().GroupKind().String(), obj.GroupVersionKind().Version)
		if err != nil {
			return err
		}

		if err = dyn.Resource(*gvr).Namespace(obj.GetNamespace()).Delete(ctx, obj.GetName(), metav1.DeleteOptions{}); err != nil {
			return err
		}
	}
	return nil
}
