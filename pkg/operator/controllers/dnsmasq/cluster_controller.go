package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

const (
	ClusterControllerName = "DnsmasqCluster"

	controllerEnabled = "aro.dnsmasq.enabled"
)

type ClusterReconciler struct {
	base.AROController
	dh dynamichelper.Interface
}

func NewClusterReconciler(log *logrus.Entry, client client.Client, dh dynamichelper.Interface) *ClusterReconciler {
	return &ClusterReconciler{
		AROController: base.AROController{
			Log:    log,
			Client: client,
			Name:   ClusterControllerName,
		},
		dh: dh,
	}
}

// Reconcile watches the ARO object, and if it changes, reconciles all the
// 99-%s-aro-dns machineconfigs
func (r *ClusterReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.GetCluster(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(controllerEnabled) {
		r.Log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.Log.Debug("running")
	mcps := &mcv1.MachineConfigPoolList{}
	err = r.Client.List(ctx, mcps)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}

	err = reconcileMachineConfigs(ctx, instance, r.dh, mcps.Items...)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}

	r.ClearConditions(ctx)
	return reconcile.Result{}, nil
}

// SetupWithManager setup our mananger
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Named(ClusterControllerName).
		Complete(r)
}

func reconcileMachineConfigs(ctx context.Context, instance *arov1alpha1.Cluster, dh dynamichelper.Interface, mcps ...mcv1.MachineConfigPool) error {
	var resources []kruntime.Object
	for _, mcp := range mcps {
		resource, err := dnsmasqMachineConfig(instance.Spec.Domain, instance.Spec.APIIntIP, instance.Spec.IngressIP, mcp.Name, instance.Spec.GatewayDomains, instance.Spec.GatewayPrivateEndpointIP)
		if err != nil {
			return err
		}

		err = dynamichelper.SetControllerReferences([]kruntime.Object{resource}, &mcp)
		if err != nil {
			return err
		}

		resources = append(resources, resource)
	}

	err := dynamichelper.Prepare(resources)
	if err != nil {
		return err
	}

	return dh.Ensure(ctx, resources...)
}
