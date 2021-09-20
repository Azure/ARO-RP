package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// Reconciler the point of the workaround controller is to apply
// workarounds that we have unitl upstream fixes are available.
type Reconciler struct {
	log *logrus.Entry

	arocli        aroclient.Interface
	configcli     configclient.Interface
	kubernetescli kubernetes.Interface

	restConfig         *rest.Config
	workarounds        map[string]Workaround
	enabledWorkarounds map[string]Workaround
}

func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, configcli configclient.Interface, kubernetescli kubernetes.Interface, mcocli mcoclient.Interface, restConfig *rest.Config) *Reconciler {
	dh, err := dynamichelper.New(log, restConfig)
	if err != nil {
		panic(err)
	}

	workarounds := map[string]Workaround{
		"systemReserved": NewSystemReserved(log, mcocli, dh),
		"ifReload":       NewIfReload(log, kubernetescli),
	}

	return &Reconciler{
		log:           log,
		arocli:        arocli,
		configcli:     configcli,
		kubernetescli: kubernetescli,
		restConfig:    restConfig,
		workarounds:   workarounds,
		enabledWorkarounds: map[string]Workaround{
			"systemReserved": workarounds["systemReserved"],
			"ifReload":       workarounds["ifReload"],
		},
	}
}

// Reconcile makes sure that the workarounds are applied or removed as per the OpenShift version.
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.Features.ReconcileWorkaroundsController {
		return reconcile.Result{}, nil
	}

	if instance.Spec.Features.ReconcileAutoSizedNodes && _, ok := r.enabledWorkarounds["systemReserved"]; !ok {
		// remove System reserved workaround, it has been replaced by autosizing
		delete(r.enabledWorkarounds, "systemReserved")
	} else if _, ok := r.enabledWorkarounds["systemReserved"]; !ok {
		r.enabledWorkarounds["systemReserved"] = r.workarounds["systemReserved"]
	}

	clusterVersion, err := version.GetClusterVersion(ctx, r.configcli)
	if err != nil {
		r.log.Errorf("error getting the OpenShift version: %v", err)
		return reconcile.Result{}, err
	}

	for _, wa := range r.enabledWorkarounds {
		if wa.IsRequired(clusterVersion) {
			err = wa.Ensure(ctx)
		} else {
			err = wa.Remove(ctx)
		}

		if err != nil {
			r.log.Errorf("workaround %s returned error %v", wa.Name(), err)
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{RequeueAfter: time.Hour, Requeue: true}, nil
}

// SetupWithManager setup our manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}).
		Named(controllers.WorkaroundControllerName).
		Complete(r)
}
