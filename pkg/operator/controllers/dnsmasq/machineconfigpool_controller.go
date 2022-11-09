package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

const (
	MachineConfigPoolControllerName = "DnsmasqMachineConfigPool"
)

type MachineConfigPoolReconciler struct {
	log *logrus.Entry

	dh dynamichelper.Interface

	client client.Client
}

func NewMachineConfigPoolReconciler(log *logrus.Entry, client client.Client, dh dynamichelper.Interface) *MachineConfigPoolReconciler {
	return &MachineConfigPoolReconciler{
		log:    log,
		dh:     dh,
		client: client,
	}
}

// Reconcile watches MachineConfigPool objects, and if any changes,
// reconciles the associated ARO DNS MachineConfig object
func (r *MachineConfigPoolReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
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
	mcp := &mcv1.MachineConfigPool{}
	err = r.client.Get(ctx, types.NamespacedName{Name: request.Name}, mcp)
	if kerrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	}
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	isMarkedToBeDeleted := mcp.GetDeletionTimestamp() != nil
	if isMarkedToBeDeleted {
		if !controllerutil.ContainsFinalizer(mcp, MachineConfigPoolControllerName) {
			return reconcile.Result{}, nil
		}

		err = r.finalize(ctx, mcp)
		if err != nil {
			r.log.Error(err)
			return reconcile.Result{}, err
		}

		controllerutil.RemoveFinalizer(mcp, MachineConfigPoolControllerName)
		err = r.dh.Ensure(ctx, mcp)
		if err != nil {
			r.log.Error(err)
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	err = reconcileMachineConfigs(ctx, instance, r.dh, *mcp)

	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// SetupWithManager setup our mananger
func (r *MachineConfigPoolReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcv1.MachineConfigPool{}).
		Named(MachineConfigPoolControllerName).
		Complete(r)
}

func (r *MachineConfigPoolReconciler) addFinalizer(ctx context.Context, mcp *mcv1.MachineConfigPool) error {
	controllerutil.AddFinalizer(mcp, MachineConfigPoolControllerName)
	return r.dh.Ensure(ctx, mcp)
}

func (r *MachineConfigPoolReconciler) finalize(ctx context.Context, mcp *mcv1.MachineConfigPool) error {
	machineConfigName := fmt.Sprintf("99-%s-aro-dns", mcp.Name)
	return r.dh.EnsureDeleted(ctx, "MachineConfig", "", machineConfigName)
}
