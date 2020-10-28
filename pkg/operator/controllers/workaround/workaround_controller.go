package workaround

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

// WorkaroundReconciler the point of the workaround controller is to apply
// workarounds that we have unitl upstream fixes are available.
type WorkaroundReconciler struct {
	kubernetescli kubernetes.Interface
	configcli     configclient.Interface
	arocli        aroclient.AroV1alpha1Interface
	restConfig    *rest.Config
	workarounds   []Workaround
	log           *logrus.Entry
}

func NewReconciler(log *logrus.Entry, kubernetescli kubernetes.Interface, configcli configclient.Interface, mcocli mcoclient.Interface, arocli aroclient.AroV1alpha1Interface, restConfig *rest.Config) *WorkaroundReconciler {
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

func (r *WorkaroundReconciler) actualClusterVersion(ctx context.Context) (*version.Version, error) {
	cv, err := r.configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	for _, history := range cv.Status.History {
		if history.State == configv1.CompletedUpdate {
			return version.ParseVersion(history.Version)
		}
	}
	return nil, fmt.Errorf("unknown cluster version")
}

// Reconcile makes sure that the workarounds are applied or removed as per the OpenShift version.
func (r *WorkaroundReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	// TODO(mj): controller-runtime master fixes the need for this (https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/reconcile/reconcile.go#L93) but it's not yet released.
	ctx := context.Background()
	clusterVersion, err := r.actualClusterVersion(ctx)
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

// SetupWithManager setup our mananger
func (r *WorkaroundReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}).
		Named(controllers.WorkaroundControllerName).
		Complete(r)
}
