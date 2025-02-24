package machinehealthcheck

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	_ "embed"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"

	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	configv1 "github.com/openshift/api/config/v1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

//go:embed staticresources/machinehealthcheck.yaml
var machinehealthcheckYaml []byte

//go:embed staticresources/mhcremediationalert.yaml
var mhcremediationalertYaml []byte

const (
	ControllerName      string = "MachineHealthCheck"
	MHCPausedAnnotation string = "cluster.x-k8s.io/paused"
)

type Reconciler struct {
	base.AROController

	dh dynamichelper.Interface
}

func NewReconciler(log *logrus.Entry, client client.Client, dh dynamichelper.Interface) *Reconciler {
	return &Reconciler{
		AROController: base.AROController{
			Log:    log,
			Client: client,
			Name:   ControllerName,
		},
		dh: dh,
	}
}

// Reconcile watches MachineHealthCheck objects, and if any changes,
// reconciles the associated ARO MachineHealthCheck object
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.GetCluster(ctx)

	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.MachineHealthCheckEnabled) {
		r.Log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.Log.Debug("running")
	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.MachineHealthCheckManaged) {
		err := r.dh.EnsureDeleted(ctx, "MachineHealthCheck", "openshift-machine-api", "aro-machinehealthcheck")
		if err != nil {
			r.Log.Error(err)
			r.SetDegraded(ctx, err)

			return reconcile.Result{RequeueAfter: time.Hour}, err
		}

		err = r.dh.EnsureDeleted(ctx, "PrometheusRule", "openshift-machine-api", "mhc-remediation-alert")
		if err != nil {
			r.Log.Error(err)
			r.SetDegraded(ctx, err)

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

		if mhc, ok := resource.(*machinev1beta1.MachineHealthCheck); ok {
			isUpgrading, err := r.isClusterUpgrading(ctx)
			if err != nil {
				r.Log.Error(err)
				r.SetDegraded(ctx, err)

				return reconcile.Result{}, err
			}

			if isUpgrading {
				mhc.ObjectMeta.Annotations = map[string]string{
					MHCPausedAnnotation: "",
				}
			}
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

func (r *Reconciler) isClusterUpgrading(ctx context.Context) (bool, error) {
	clusterVersion := &configv1.ClusterVersion{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: "version"}, clusterVersion); err != nil {
		return false, err
	}

	for _, cnd := range clusterVersion.Status.Conditions {
		if cnd.Type == configv1.OperatorProgressing {
			return cnd.Status == configv1.ConditionTrue, nil
		}
	}

	return false, nil
}

// SetupWithManager will manage only our MHC resource with our specific controller name
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		Named(ControllerName).
		Owns(&machinev1beta1.MachineHealthCheck{}).
		Owns(&monitoringv1.PrometheusRule{}).
		Watches(
			&configv1.ClusterVersion{},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(predicates.ClusterVersion),
		).
		Complete(r)
}
