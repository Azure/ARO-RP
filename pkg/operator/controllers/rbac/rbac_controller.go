package rbac

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

type RBACReconciler struct {
	log *logrus.Entry

	arocli aroclient.Interface
	dh     dynamichelper.Interface
}

func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, dh dynamichelper.Interface) *RBACReconciler {
	return &RBACReconciler{
		log:    log,
		arocli: arocli,
		dh:     dh,
	}
}

func (r *RBACReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, request.Name, metav1.GetOptions{})
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	var resources []runtime.Object
	for _, assetName := range AssetNames() {
		b, err := Asset(assetName)
		if err != nil {
			r.log.Error(err)
			return reconcile.Result{}, err
		}

		resource, _, err := scheme.Codecs.UniversalDeserializer().Decode(b, nil, nil)
		if err != nil {
			r.log.Error(err)
			return reconcile.Result{}, err
		}

		resources = append(resources, resource)
	}

	err = dynamichelper.SetControllerReferences(resources, instance)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	err = dynamichelper.Prepare(resources)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	err = r.dh.Ensure(ctx, resources...)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// SetupWithManager setup our mananger
func (r *RBACReconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Named(controllers.RBACControllerName).
		Complete(r)
}
