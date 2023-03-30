package dnsmasq

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"regexp"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

const (
	MachineConfigControllerName = "DnsmasqMachineConfig"
)

type MachineConfigReconciler struct {
	log *logrus.Entry

	dh dynamichelper.Interface

	client client.Client
}

var rxARODNS = regexp.MustCompile("^99-(.*)-aro-dns$")

func NewMachineConfigReconciler(log *logrus.Entry, client client.Client, dh dynamichelper.Interface) *MachineConfigReconciler {
	return &MachineConfigReconciler{
		log:    log,
		dh:     dh,
		client: client,
	}
}

// Reconcile watches ARO DNS MachineConfig objects, and if any changes,
// reconciles it
func (r *MachineConfigReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
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
	m := rxARODNS.FindStringSubmatch(request.Name)
	if m == nil {
		return reconcile.Result{}, nil
	}
	role := m[1]

	mcp := &mcv1.MachineConfigPool{}
	err = r.client.Get(ctx, types.NamespacedName{Name: role}, mcp)
	if kerrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	}
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	err = reconcileMachineConfigs(ctx, instance, r.dh, role)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// SetupWithManager setup our mananger
func (r *MachineConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcv1.MachineConfig{}).
		Named(MachineConfigControllerName).
		Complete(r)
}
