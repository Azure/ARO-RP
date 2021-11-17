package checker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	maoclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
)

const (
	CONFIG_NAMESPACE string = "aro.checker"
	ENABLED          string = CONFIG_NAMESPACE + ".enabled"
)

// Reconciler runs a number of checkers
type Reconciler struct {
	log *logrus.Entry

	role     string
	checkers []Checker
	arocli   aroclient.Interface
}

func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, kubernetescli kubernetes.Interface, maocli maoclient.Interface, role string) *Reconciler {
	checkers := []Checker{NewInternetChecker(log, arocli, role)}

	if role == operator.RoleMaster {
		checkers = append(checkers,
			NewServicePrincipalChecker(log, arocli, kubernetescli, maocli, role),
		)
	}

	return &Reconciler{
		log:      log,
		role:     role,
		checkers: checkers,
		arocli:   arocli,
	}
}

// This is the permissions that this controller needs to work.
// "make generate" will run kubebuilder and cause operator/deploy/staticresources/*/role.yaml to be updated
// from the annotation below.
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=aro.openshift.io,resources=clusters/status,verbs=get;update;patch

// Reconcile will keep checking that the cluster can connect to essential services.
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(ENABLED) {
		// controller is disabled
		return reconcile.Result{}, nil
	}

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
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})

	builder := ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate))

	return builder.Named(controllers.CheckerControllerName).Complete(r)
}
