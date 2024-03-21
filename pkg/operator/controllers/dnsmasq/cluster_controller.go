package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

const (
	ClusterControllerName = "DnsmasqCluster"
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

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.DnsmasqEnabled) {
		r.Log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	restartDnsmasq := instance.Spec.OperatorFlags.GetSimpleBoolean(operator.RestartDnsmasqEnabled)
	if restartDnsmasq {
		r.Log.Debug("restartDnsmasq is enabled")
	}

	allowReconcile, err := r.AllowRebootCausingReconciliation(ctx, instance)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}

	if !allowReconcile {
		r.Log.Debugf("skipping reconciliation of %s", r.Name)
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

	err = reconcileMachineConfigs(ctx, instance, r.dh, r.Client, allowReconcile, restartDnsmasq, mcps.Items...)
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

func reconcileMachineConfigs(ctx context.Context, instance *arov1alpha1.Cluster, dh dynamichelper.Interface, c client.Client, allowReconcile bool, restartDnsmasq bool, mcps ...mcv1.MachineConfigPool) error {
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

	// If we are allowed to reconcile the resources, then we run Ensure to
	// create or update. If we are not allowed to reconcile, we do not want to
	// perform any updates, but we do want to perform initial configuration.
	if allowReconcile {
		return dh.Ensure(ctx, resources...)
	} else {
		for _, i := range resources {
			err := c.Create(ctx, i.(client.Object))
			// Since we are only creating, ignore AlreadyExists
			if err != nil && !kerrors.IsAlreadyExists(err) {
				return fmt.Errorf("error creating client object: %w", err)
			}
		}
	}
	return nil
}
