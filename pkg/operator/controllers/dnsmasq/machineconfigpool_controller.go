package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	mcoclient "github.com/openshift/machine-config-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

type MachineConfigPoolReconciler struct {
	log *logrus.Entry

	arocli aroclient.Interface
	mcocli mcoclient.Interface
	dh     dynamichelper.Interface
}

func NewMachineConfigPoolReconciler(log *logrus.Entry, arocli aroclient.Interface, mcocli mcoclient.Interface, dh dynamichelper.Interface) *MachineConfigPoolReconciler {
	return &MachineConfigPoolReconciler{
		log:    log,
		arocli: arocli,
		mcocli: mcocli,
		dh:     dh,
	}
}

// Reconcile watches MachineConfigPool objects, and if any changes,
// reconciles the associated ARO DNS MachineConfig object
func (r *MachineConfigPoolReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	_, err := r.mcocli.MachineconfigurationV1().MachineConfigPools().Get(ctx, request.Name, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	}
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	err = reconcileMachineConfigs(ctx, r.arocli, r.dh, request.Name)
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
		Named(controllers.DnsmasqMachineConfigPoolControllerName).
		Complete(r)
}
