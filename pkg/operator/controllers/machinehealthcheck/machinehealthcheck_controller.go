package machinehealthcheck

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"
	"time"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

const (
	ControllerName string = "MachineHealthCheck"
	managed        string = "aro.machinehealthcheck.managed"
	enabled        string = "aro.machinehealthcheck.enabled"
)

type Reconciler struct {
	arocli aroclient.Interface
	dh     dynamichelper.Interface
}

func NewReconciler(arocli aroclient.Interface, dh dynamichelper.Interface) *Reconciler {
	return &Reconciler{
		arocli: arocli,
		dh:     dh,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(enabled) {
		return reconcile.Result{}, nil
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(managed) {
		err := r.dh.EnsureDeleted(ctx, "MachineHealthCheck", "openshift-machine-api", "aro-machinehealthcheck")
		if err != nil {
			return reconcile.Result{RequeueAfter: time.Hour}, err
		}

		err = r.dh.EnsureDeleted(ctx, "PrometheusRule", "openshift-machine-api", "mhc-remediation-alert")
		if err != nil {
			return reconcile.Result{RequeueAfter: time.Hour}, err
		}
		return reconcile.Result{}, nil
	}

	var resources []kruntime.Object

	// this loop prevents us from hard coding resource strings
	// and ensures all static resources are accounted for.
	for _, assetName := range AssetNames() {
		b, err := Asset(assetName)
		if err != nil {
			return reconcile.Result{}, err
		}

		resource, _, err := scheme.Codecs.UniversalDeserializer().Decode(b, nil, nil)
		if err != nil {
			return reconcile.Result{}, err
		}

		resources = append(resources, resource)
	}

	// helps with garbage collection of the resources we are dealing with
	err = dynamichelper.SetControllerReferences(resources, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// make sure we will be able to deploy a new resource into the cluster
	err = dynamichelper.Prepare(resources)
	if err != nil {
		return reconcile.Result{}, err
	}

	// create/update the MHC CR
	err = r.dh.Ensure(ctx, resources...)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// SetupWithManager will manage only our MHC resource with our specific controller name
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return strings.EqualFold(arov1alpha1.SingletonClusterName, o.GetName())
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Named(ControllerName).
		Owns(&machinev1beta1.MachineHealthCheck{}).
		Owns(&monitoringv1.PrometheusRule{}).
		Complete(r)
}
