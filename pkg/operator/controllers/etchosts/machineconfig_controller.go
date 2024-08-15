package etchosts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"regexp"

	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/base"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

const (
	EtcHostsMachineConfigControllerName = "DnsmasqMachineConfig"
)

type MachineConfigReconciler struct {
	base.AROController

	dh dynamichelper.Interface
}

var etcHostsRegex = regexp.MustCompile("^99-(.*)-aro-etc-hosts-gateway-domains$")

func NewMachineConfigReconciler(log *logrus.Entry, client client.Client, dh dynamichelper.Interface) *MachineConfigReconciler {
	return &MachineConfigReconciler{
		AROController: base.AROController{
			Log:    log,
			Client: client,
			Name:   EtcHostsMachineConfigControllerName,
		},
		dh: dh,
	}
}

// Reconcile watches ARO DNS MachineConfig objects, and if any changes,
// reconciles it
func (r *MachineConfigReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.GetCluster(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.EtcHostsMachineConfigEnabled) {
		r.Log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.Log.Debug("running")
	m := etcHostsRegex.FindStringSubmatch(request.Name)
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

	err = reconcileMachineConfigs(ctx, instance, role, r.dh, *mcp)
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
		Named(EtcHostsMachineConfigControllerName).
		Complete(r)
}

func reconcileMachineConfigs(ctx context.Context, instance *arov1alpha1.Cluster, role string, dh dynamichelper.Interface, mcps ...mcv1.MachineConfigPool) error {
	var resources []kruntime.Object
	for _, mcp := range mcps {
		resource, err := EtcHostsMachineConfig(instance.Spec.Domain, instance.Spec.APIIntIP, instance.Spec.GatewayDomains, instance.Spec.GatewayPrivateEndpointIP, role)
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
