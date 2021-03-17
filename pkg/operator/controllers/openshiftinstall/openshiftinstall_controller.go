package openshiftinstall

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator/controllers"
)

var openshiftInstallConfigMap = types.NamespacedName{Name: "openshift-install-manifests", Namespace: "openshift-config"}

// OpenshiftInstallReconciler reconciles the openshift-install-manifest ConfigMap
type OpenshiftInstallReconciler struct {
	kubernetescli kubernetes.Interface
	log           *logrus.Entry
}

func NewReconciler(log *logrus.Entry, kubernetescli kubernetes.Interface) *OpenshiftInstallReconciler {
	return &OpenshiftInstallReconciler{
		kubernetescli: kubernetescli,
		log:           log,
	}
}

// Reconcile makes sure that the openshift-install-manifest ConfigMap has the right value for invoker key.
func (r *OpenshiftInstallReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	return reconcile.Result{}, r.setOpenshiftInstallInvoker(ctx, "AR0")
}

// setOpenshiftInstallInvoker sets data.invoker for openshift-install-manifest to invoker string. This will be picked up by cluster-version-operator to generate cluster_installer metric
// and then be taken by Telemetry
func (r *OpenshiftInstallReconciler) setOpenshiftInstallInvoker(ctx context.Context, invoker string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		oim, err := r.kubernetescli.CoreV1().ConfigMaps(openshiftInstallConfigMap.Namespace).Get(ctx, openshiftInstallConfigMap.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if inv, ok := oim.Data["invoker"]; ok {
			if inv == invoker {
				// Invoker already has the right value, no need for update
				return nil
			}
		}

		oim.Data["invoker"] = invoker

		_, err = r.kubernetescli.CoreV1().ConfigMaps(openshiftInstallConfigMap.Namespace).Update(ctx, oim, metav1.UpdateOptions{})
		return err
	})
}

// SetupWithManager setup reconcilier with manager to reconcile openshift-install-manifest ConfigMap
func (r *OpenshiftInstallReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.log.Info("Setting openshift-install-manifest invoker to ARO")

	isOpenshiftInstallPredicate := predicate.NewPredicateFuncs(func(meta metav1.Object, object runtime.Object) bool {
		return meta.GetName() == openshiftInstallConfigMap.Name && meta.GetNamespace() == openshiftInstallConfigMap.Namespace
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}, builder.WithPredicates(isOpenshiftInstallPredicate)).
		Named(controllers.OpenshiftInstallControllerName).
		Complete(r)
}
