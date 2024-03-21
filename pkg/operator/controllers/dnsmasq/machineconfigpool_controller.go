package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

const (
	MachineConfigPoolControllerName = "DnsmasqMachineConfigPool"
)

type MachineConfigPoolReconciler struct {
	base.AROController

	dh dynamichelper.Interface
}

func NewMachineConfigPoolReconciler(log *logrus.Entry, client client.Client, dh dynamichelper.Interface) *MachineConfigPoolReconciler {
	return &MachineConfigPoolReconciler{
		AROController: base.AROController{
			Log:    log,
			Client: client,
			Name:   MachineConfigPoolControllerName,
		},
		dh: dh,
	}
}

// Reconcile watches MachineConfigPool objects, and if any changes,
// reconciles the associated ARO DNS MachineConfig object
func (r *MachineConfigPoolReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
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
		r.Log.Debug("restart dnsmasq machineconfig enabled")
	}

	allowReconcile, err := r.AllowRebootCausingReconciliation(ctx, instance)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}

	r.Log.Debug("running")
	mcp := &mcv1.MachineConfigPool{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: request.Name}, mcp)
	if kerrors.IsNotFound(err) {
		r.ClearDegraded(ctx)
		return reconcile.Result{}, nil
	}
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}

	err = reconcileMachineConfigs(ctx, instance, r.dh, r.Client, allowReconcile, restartDnsmasq, *mcp)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}

	r.ClearConditions(ctx)
	return reconcile.Result{}, nil
}

// SetupWithManager setup our mananger
func (r *MachineConfigPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcv1.MachineConfigPool{}).
		Named(MachineConfigPoolControllerName).
		Complete(r)
}
