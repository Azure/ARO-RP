package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"time"

	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maoclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	"github.com/Azure/ARO-RP/pkg/util/deployment"
)

// CheckerController runs a number of checkers
type CheckerController struct {
	log      *logrus.Entry
	role     string
	checkers []Checker
}

func NewReconciler(log *logrus.Entry, clustercli maoclient.Interface, arocli aroclient.AroV1alpha1Interface, role string, deploymentMode deployment.Mode) *CheckerController {
	checkers := []Checker{NewInternetChecker(log, arocli, role)}

	if role == operator.RoleMaster {
		checkers = append(checkers, NewMachineChecker(log, clustercli, arocli, role, deploymentMode))
	}

	return &CheckerController{
		log:      log,
		role:     role,
		checkers: checkers,
	}
}

// This is the permissions that this controller needs to work.
// "make generate" will run kubebuilder and cause operator/deploy/staticresources/*/role.yaml to be updated
// from the annotation below.
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters/status,verbs=get;update;patch

// Reconcile will keep checking that the cluster can connect to essential services.
func (r *CheckerController) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	var err error
	for _, c := range r.checkers {
		thisErr := c.Check()
		if thisErr != nil {
			// do all checks even if there is an error
			err = thisErr
			r.log.Errorf("checker %s failed with %v", c.Name(), err)
		}
	}

	return reconcile.Result{RequeueAfter: time.Hour, Requeue: true}, err
}

// SetupWithManager setup our mananger
func (r *CheckerController) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).For(&arov1alpha1.Cluster{})
	if r.role == operator.RoleMaster {
		// https://github.com/kubernetes-sigs/controller-runtime/issues/1173
		// equivalent to builder = builder.For(&machinev1beta1.Machine{}), but can't call For multiple times on one builder
		builder = builder.Watches(&source.Kind{Type: &machinev1beta1.Machine{}}, &handler.EnqueueRequestForObject{})
	}
	return builder.Named(controllers.CheckerControllerName).Complete(r)
}
