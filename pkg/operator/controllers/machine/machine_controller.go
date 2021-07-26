package machine

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strings"

	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maoclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
	"github.com/operator-framework/operator-sdk/pkg/status"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	_ "github.com/Azure/ARO-RP/pkg/util/scheme"
)

type Reconciler struct {
	log *logrus.Entry

	arocli aroclient.Interface
	maocli maoclient.Interface

	isLocalDevelopmentMode bool
	role                   string
}

func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, maocli maoclient.Interface, isLocalDevelopmentMode bool, role string) *Reconciler {
	return &Reconciler{
		log:                    log,
		arocli:                 arocli,
		maocli:                 maocli,
		isLocalDevelopmentMode: isLocalDevelopmentMode,
		role:                   role,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	// Update cluster object's status.
	cond := &status.Condition{
		Type:    arov1alpha1.MachineValid,
		Status:  corev1.ConditionTrue,
		Message: "All machines valid",
		Reason:  "CheckDone",
	}

	errs := r.checkMachines(ctx)
	if len(errs) > 0 {
		cond.Status = corev1.ConditionFalse
		cond.Reason = "CheckFailed"

		var sb strings.Builder
		for _, err := range errs {
			sb.WriteString(err.Error())
			sb.WriteByte('\n')
		}
		cond.Message = sb.String()
	}

	return reconcile.Result{}, controllers.SetCondition(ctx, r.arocli, cond, r.role)
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&machinev1beta1.Machine{}).
		Named(controllers.MachineControllerName).
		Complete(r)
}
