package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maoclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
)

// CheckerController runs a number of checkers
type CheckerController struct {
	log      *logrus.Entry
	role     string
	checkers []Checker
}

func NewReconciler(log *logrus.Entry, maocli maoclient.Interface, arocli aroclient.Interface, kubernetescli kubernetes.Interface, role string, isLocalDevelopmentMode bool) *CheckerController {
	checkers := []Checker{NewInternetChecker(log, arocli, role)}

	if role == operator.RoleMaster {
		checkers = append(checkers,
			NewMachineChecker(log, maocli, arocli, role, isLocalDevelopmentMode),
			NewServicePrincipalChecker(log, maocli, arocli, kubernetescli, role),
		)
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
	// TODO(mj): Reconcile will eventually be receiving a ctx (https://github.com/kubernetes-sigs/controller-runtime/blob/7ef2da0bc161d823f084ad21ff5f9c9bd6b0cc39/pkg/reconcile/reconcile.go#L93)
	ctx := context.TODO()

	var err error
	for _, c := range r.checkers {
		thisErr := c.Check(ctx)
		if thisErr != nil {
			// do all checks even if there is an error
			err = thisErr
			if thisErr != errRequeue {
				r.log.Errorf("checker %s failed with %v", c.Name(), err)
			}
		}
	}

	return reconcile.Result{RequeueAfter: time.Hour, Requeue: true}, err
}

// SetupWithManager setup our manager
func (r *CheckerController) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(meta metav1.Object, object runtime.Object) bool {
		return meta.GetName() == arov1alpha1.SingletonClusterName
	})

	builder := ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate))

	if r.role == operator.RoleMaster {
		// https://github.com/kubernetes-sigs/controller-runtime/issues/1173
		// equivalent to builder = builder.For(&machinev1beta1.Machine{}), but can't call For multiple times on one builder
		builder = builder.Watches(&source.Kind{Type: &machinev1beta1.Machine{}}, &handler.EnqueueRequestForObject{})
	}
	return builder.Named(controllers.CheckerControllerName).Complete(r)
}
