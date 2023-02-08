package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/openshift/installer/pkg/aro/dnsmasq"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

const (
	ClusterControllerName = "DnsmasqCluster"

	controllerEnabled = "aro.dnsmasq.enabled"
)

type ClusterReconciler struct {
	log *logrus.Entry

	dh dynamichelper.Interface

	client client.Client
}

func NewClusterReconciler(log *logrus.Entry, client client.Client, dh dynamichelper.Interface) *ClusterReconciler {
	return &ClusterReconciler{
		log:    log,
		dh:     dh,
		client: client,
	}
}

// Reconcile watches the ARO object, and if it changes, reconciles all the
// 99-%s-aro-dns machineconfigs
func (r *ClusterReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance := &arov1alpha1.Cluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(controllerEnabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running")
	mcps := &mcv1.MachineConfigPoolList{}
	err = r.client.List(ctx, mcps)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	roles := make([]string, 0, len(mcps.Items))
	for _, mcp := range mcps.Items {
		roles = append(roles, mcp.Name)
	}

	err = reconcileMachineConfigs(ctx, instance, r.dh, roles...)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func reconcileMachineConfigs(ctx context.Context, instance *arov1alpha1.Cluster, dh dynamichelper.Interface, roles ...string) error {
	var resources []kruntime.Object
	for _, role := range roles {
		resource, err := dnsmasq.MachineConfig(instance.Spec.Domain, instance.Spec.APIIntIP, instance.Spec.IngressIP, role, instance.Spec.GatewayDomains, instance.Spec.GatewayPrivateEndpointIP)
		if err != nil {
			return err
		}

		resources = append(resources, resource)
	}

	err := dynamichelper.SetControllerReferences(resources, instance)
	if err != nil {
		return err
	}

	err = dynamichelper.Prepare(resources)
	if err != nil {
		return err
	}

	return dh.Ensure(ctx, resources...)
}

// SetupWithManager setup our manager
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Named(ClusterControllerName).
		Complete(r)
}
