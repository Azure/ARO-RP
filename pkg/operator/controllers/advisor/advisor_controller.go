package advisor

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"time"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	"github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
)

// Reconciler runs a number of checkers
type Reconciler struct {
	log *logrus.Entry

	role     string
	checkers []Advisor
}

func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, operatorcli operatorclient.Interface, configcli configclient.Interface, role string) *Reconciler {
	checkers := []Advisor{
		NewIngressCertificateChecker(log, arocli, operatorcli, configcli, role),
	}

	return &Reconciler{
		log:      log,
		role:     role,
		checkers: checkers,
	}
}

// Reconcile will keep checking the configuration of cluster services
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	var err error
	for _, c := range r.checkers {
		thisErr := c.Check(ctx)
		if thisErr != nil {
			// do all checks even if there is an error
			err = thisErr
			if thisErr != errRequeue {
				r.log.Errorf("advisor checker %s failed with %v", c.Name(), err)
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
