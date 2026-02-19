package dns

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"regexp"

	"github.com/sirupsen/logrus"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
)

const (
	MachineConfigControllerName = "DnsmasqMachineConfig"
)

type MachineConfigReconciler struct {
	base.AROController

	ch clienthelper.Interface
}

var rxARODNS = regexp.MustCompile("^99-(.*)-aro-dns$")

func NewMachineConfigReconciler(log *logrus.Entry, client client.Client, ch clienthelper.Interface) *MachineConfigReconciler {
	return &MachineConfigReconciler{
		AROController: base.AROController{
			Log:    log,
			Client: client,
			Name:   MachineConfigControllerName,
		},
		ch: ch,
	}
}

// Reconcile watches ARO DNS MachineConfig objects, and if any changes,
// reconciles it
func (r *MachineConfigReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.GetCluster(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !IsDNSControllerEnabled(instance.Spec.OperatorFlags) {
		r.Log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	// CustomDNS clusters don't use 99-*-aro-dns MachineConfigs; skip reconciliation
	if GetEffectiveDNSType(ctx, r.Client, r.Log, instance) == operator.DNSTypeClusterHosted {
		r.Log.Debug("CustomDNS enabled, skipping dnsmasq MachineConfig reconciliation")
		return reconcile.Result{}, nil
	}

	if instance.Spec.OperatorFlags.GetSimpleBoolean(operator.RestartDnsmasqEnabled) {
		r.Log.Debug("restart dnsmasq machineconfig enabled")
	}

	allowReconcile, err := r.AllowRebootCausingReconciliation(ctx, instance)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}

	r.Log.Debug("running")
	m := rxARODNS.FindStringSubmatch(request.Name)
	if m == nil {
		return reconcile.Result{}, nil
	}
	role := m[1]

	mcp := &mcv1.MachineConfigPool{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: role}, mcp)
	if kerrors.IsNotFound(err) {
		r.ClearDegraded(ctx)
		return reconcile.Result{}, nil
	}
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}
	if mcp.GetDeletionTimestamp() != nil {
		return reconcile.Result{}, nil
	}

	err = reconcileMachineConfigs(ctx, instance, r.ch, r.Client, allowReconcile, instance.Spec.OperatorFlags.GetSimpleBoolean(operator.RestartDnsmasqEnabled), *mcp)
	if err != nil {
		r.Log.Error(err)
		r.SetDegraded(ctx, err)
		return reconcile.Result{}, err
	}

	r.ClearConditions(ctx)
	return reconcile.Result{}, nil
}

// SetupWithManager setup our mananger
func (r *MachineConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcv1.MachineConfig{}).
		Named(MachineConfigControllerName).
		Complete(r)
}
