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

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

const (
	clusterControllerName = "DnsmasqCluster"
	controllerEnabled     = operator.DnsmasqEnabled // calling from flags.go due to removal on #3327
	restartDnsmasqEnabled = operator.RestartDnsmasqEnabled
)

type ClusterReconciler struct {
	base.AROController
	dh dynamichelper.Interface
}

func NewClusterReconciler(log *logrus.Entry, client client.Client, dh dynamichelper.Interface) *ClusterReconciler {
	r := &ClusterReconciler{
		AROController: base.AROController{
			Log:         log.WithField("controller", clusterControllerName),
			Client:      client,
			Name:        clusterControllerName,
			EnabledFlag: controllerEnabled,
		},
		dh: dh,
	}
	r.Reconciler = r
	return r
}

// Reconcile watches the ARO object, and if it changes, reconciles all the
// 99-%s-aro-dns machineconfigs
func (r *ClusterReconciler) ReconcileEnabled(ctx context.Context, request ctrl.Request, instance *arov1alpha1.Cluster) (ctrl.Result, error) {
	var err error

	restartDnsmasq := instance.Spec.OperatorFlags.GetSimpleBoolean(operator.RestartDnsmasqEnabled)
	if restartDnsmasq {
		r.Log.Debug("restartDnsmasq is enabled")
	}

	mcps := &mcv1.MachineConfigPoolList{}
	err = r.Client.List(ctx, mcps)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}

	err = reconcileMachineConfigs(ctx, instance, r.dh, restartDnsmasq, mcps.Items...)
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
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		Named(r.GetName()).
		Complete(r)
}

func reconcileMachineConfigs(ctx context.Context, instance *arov1alpha1.Cluster, dh dynamichelper.Interface, restartDnsmasq bool, mcps ...mcv1.MachineConfigPool) error {
	var resources []kruntime.Object
	for _, mcp := range mcps {
		resource, err := dnsmasqMachineConfig(instance.Spec.Domain, instance.Spec.APIIntIP, instance.Spec.IngressIP, mcp.Name, instance.Spec.GatewayDomains, instance.Spec.GatewayPrivateEndpointIP, restartDnsmasq)
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
