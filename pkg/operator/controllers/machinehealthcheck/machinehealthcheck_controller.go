package machinehealthcheck

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	_ "embed"
	"strings"
	"time"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

//go:embed staticresources/machinehealthcheck.yaml
var machinehealthcheckYaml []byte

//go:embed staticresources/mhcremediationalert.yaml
var mhcremediationalertYaml []byte

const (
	MHCControllerName string = "MachineHealthCheck"
	MHCManaged        string = "aro.machinehealthcheck.managed"
	MHCEnabled        string = "aro.machinehealthcheck.enabled"
)

type MachineHealthCheckReconciler struct {
	base.AROController

	dh dynamichelper.Interface
}

func NewMachineHealthCheckReconciler(log *logrus.Entry, client client.Client, dh dynamichelper.Interface) *MachineHealthCheckReconciler {
	return &MachineHealthCheckReconciler{
		AROController: base.AROController{
			Log:    log,
			Client: client,
			Name:   MHCControllerName,
		},
		dh: dh,
	}
}

// Reconcile watches MachineHealthCheck objects, and if any changes,
// reconciles the associated ARO MachineHealthCheck object
func (r *MachineHealthCheckReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.GetCluster(ctx)

	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(MHCEnabled) {
		r.Log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.Log.Debug("running")
	if !instance.Spec.OperatorFlags.GetSimpleBoolean(MHCManaged) {
		r.SetProgressing(ctx, "MHC and it's alerts are disabled because this cluster is not MHC managed.")

		err := r.dh.EnsureDeleted(ctx, "MachineHealthCheck", "openshift-machine-api", "aro-machinehealthcheck")
		if err != nil {
			r.Log.Error(err)
			r.SetDegraded(ctx, err)
			r.ClearProgressing(ctx)

			return reconcile.Result{RequeueAfter: time.Hour}, err
		}

		err = r.dh.EnsureDeleted(ctx, "PrometheusRule", "openshift-machine-api", "mhc-remediation-alert")
		if err != nil {
			r.Log.Error(err)
			r.SetDegraded(ctx, err)
			r.ClearProgressing(ctx)

			return reconcile.Result{RequeueAfter: time.Hour}, err
		}

		r.ClearConditions(ctx)
		return reconcile.Result{}, nil
	}

	var resources []kruntime.Object

	for _, asset := range [][]byte{machinehealthcheckYaml, mhcremediationalertYaml} {
		resource, _, err := scheme.Codecs.UniversalDeserializer().Decode(asset, nil, nil)
		if err != nil {
			r.Log.Error(err)
			r.SetDegraded(ctx, err)

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
		r.Log.Error(err)
		r.SetDegraded(ctx, err)

		return reconcile.Result{}, err
	}

	r.ClearConditions(ctx)
	return reconcile.Result{}, nil
}

// SetupWithManager will manage only our MHC resource with our specific controller name
func (r *MachineHealthCheckReconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return strings.EqualFold(arov1alpha1.SingletonClusterName, o.GetName())
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Named(MHCControllerName).
		Owns(&machinev1beta1.MachineHealthCheck{}).
		Owns(&monitoringv1.PrometheusRule{}).
		Complete(r)
}
