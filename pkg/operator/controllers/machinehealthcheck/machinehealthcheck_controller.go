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
	corev1 "k8s.io/api/core/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

//go:embed staticresources/machinehealthcheck.yaml
var machinehealthcheckYaml []byte

//go:embed staticresources/mhcremediationalert.yaml
var mhcremediationalertYaml []byte

const (
	ControllerName    string = "MachineHealthCheck"
	managed           string = "aro.machinehealthcheck.managed"
	enabled           string = "aro.machinehealthcheck.enabled"
	maxSupportedNodes int    = 62
)

type Reconciler struct {
	log *logrus.Entry

	dh dynamichelper.Interface

	client client.Client
}

func NewReconciler(log *logrus.Entry, client client.Client, dh dynamichelper.Interface) *Reconciler {
	return &Reconciler{
		log:    log,
		dh:     dh,
		client: client,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance := &arov1alpha1.Cluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(enabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running")
	nodeList := &corev1.NodeList{}
	err = r.client.List(ctx, nodeList)
	if err != nil {
		return reconcile.Result{}, err
	}
	removeMHCCR := len(nodeList.Items) == maxSupportedNodes

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(managed) || removeMHCCR {
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

	for _, asset := range [][]byte{machinehealthcheckYaml, mhcremediationalertYaml} {
		resource, _, err := scheme.Codecs.UniversalDeserializer().Decode(asset, nil, nil)
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
		Watches(&source.Kind{Type: &corev1.Node{}}, &handler.EnqueueRequestForObject{}).
		Named(ControllerName).
		Owns(&machinev1beta1.MachineHealthCheck{}).
		Owns(&monitoringv1.PrometheusRule{}).
		Complete(r)
}
