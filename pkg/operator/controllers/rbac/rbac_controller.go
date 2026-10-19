package rbac

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io/fs"

	"github.com/sirupsen/logrus"

	rbacv1 "k8s.io/api/rbac/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

const (
	ControllerName = "RBAC"
)

type Reconciler struct {
	log *logrus.Entry

	ch clienthelper.Interface
}

func NewReconciler(log *logrus.Entry, ch clienthelper.Interface) *Reconciler {
	return &Reconciler{
		log: log,
		ch:  ch,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance := &arov1alpha1.Cluster{}
	err := r.ch.GetOne(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.RbacEnabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running")
	var resources []kruntime.Object
	fserr := fs.WalkDir(staticResources, "staticresources", func(path string, d fs.DirEntry, _ error) error {
		// ignore the error being passed in as it is only for directory read
		// failures (which as this is a static fs, should not happen anyway)

		// We only care about the files
		if d.IsDir() {
			return nil
		}

		b, err := staticResources.ReadFile(path)
		if err != nil {
			return err
		}

		resource, _, err := scheme.Codecs.UniversalDeserializer().Decode(b, nil, nil)
		if err != nil {
			return err
		}

		resources = append(resources, resource)
		return nil
	})
	if fserr != nil {
		r.log.Error(fserr)
		return reconcile.Result{}, fserr
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

	err = r.ch.Ensure(ctx, resources...)
	if err != nil {
		r.log.Error(err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// SetupWithManager setup our mananger
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Named(ControllerName).
		Complete(r)
}
