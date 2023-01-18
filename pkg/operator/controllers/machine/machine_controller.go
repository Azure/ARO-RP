package machine

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/conditions"
)

const (
	ControllerName = "Machine"

	controllerEnabled = "aro.machine.enabled"
)

type Reconciler struct {
	log *logrus.Entry

	isLocalDevelopmentMode bool
	role                   string

	client client.Client
}

func NewReconciler(log *logrus.Entry, client client.Client, isLocalDevelopmentMode bool, role string) *Reconciler {
	return &Reconciler{
		log:                    log,
		isLocalDevelopmentMode: isLocalDevelopmentMode,
		role:                   role,
		client:                 client,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
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
	// Update cluster object's status.
	cond := &operatorv1.OperatorCondition{
		Type:    arov1alpha1.MachineValid,
		Status:  operatorv1.ConditionTrue,
		Message: "All machines valid",
		Reason:  "CheckDone",
	}

	errs := r.checkMachines(ctx)
	if len(errs) > 0 {
		cond.Status = operatorv1.ConditionFalse
		cond.Reason = "CheckFailed"

		var sb strings.Builder
		for _, err := range errs {
			sb.WriteString(err.Error())
			sb.WriteByte('\n')
		}
		cond.Message = sb.String()
	}

	return reconcile.Result{}, conditions.SetCondition(ctx, r.client, cond, r.role)
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&machinev1beta1.Machine{}).
		Named(ControllerName).
		Complete(r)
}
