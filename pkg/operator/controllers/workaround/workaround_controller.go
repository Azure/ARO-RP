package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
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

// WorkaroundReconciler the point of the workaround controller is to apply
// workarounds that we have unitl upstream fixes are available.
type WorkaroundReconciler struct {
	kubernetescli kubernetes.Interface
	configcli     configclient.Interface
	arocli        aroclient.Interface
	restConfig    *rest.Config
	workarounds   []Workaround
	log           *logrus.Entry
}

func NewReconciler(log *logrus.Entry, kubernetescli kubernetes.Interface, configcli configclient.Interface, mcocli mcoclient.Interface, arocli aroclient.Interface, restConfig *rest.Config) *WorkaroundReconciler {
	dh, err := dynamichelper.New(log, restConfig)
	if err != nil {
		panic(err)
	}

	return &WorkaroundReconciler{
		kubernetescli: kubernetescli,
		configcli:     configcli,
		arocli:        arocli,
		restConfig:    restConfig,
		workarounds:   []Workaround{NewSystemReserved(log, mcocli, dh), NewIfReload(log, kubernetescli)},
		log:           log,
	}
}

// Reconcile makes sure that the workarounds are applied or removed as per the OpenShift version.
func (r *WorkaroundReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	clusterVersion, err := version.GetClusterVersion(ctx, r.configcli)
	if err != nil {
		r.log.Errorf("error getting the OpenShift version: %v", err)
		return reconcile.Result{}, err
	}

	for _, wa := range r.workarounds {
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
func (r *WorkaroundReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}).
		Named(controllers.WorkaroundControllerName).
		Complete(r)
}
