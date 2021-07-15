package machine

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	operatorv1 "github.com/openshift/api/operator/v1"
	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maoclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	"github.com/Azure/ARO-RP/pkg/util/conditions"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

type MachineReconciler struct {
	maocli                 maoclient.Interface
	arocli                 aroclient.Interface
	log                    *logrus.Entry
	isLocalDevelopmentMode bool
	role                   string
}

func NewMachineReconciler(log *logrus.Entry, maocli maoclient.Interface, arocli aroclient.Interface, isLocalDevelopmentMode bool, role string) *MachineReconciler {
	return &MachineReconciler{
		maocli:                 maocli,
		arocli:                 arocli,
		log:                    log,
		isLocalDevelopmentMode: isLocalDevelopmentMode,
		role:                   role,
	}
}

func (r *MachineReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	// Update cluster object's status.
	cond := &operatorv1.OperatorCondition{
		Type:    arov1alpha1.MachineValid.String(),
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

	return reconcile.Result{}, conditions.SetCondition(ctx, r.arocli, cond, r.role)
}

func (r *MachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&machinev1beta1.Machine{}).
		Named(controllers.MachineControllerName).
		Complete(r)
}
