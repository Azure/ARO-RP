package previewfeature

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	aropreviewv1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/preview.aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/previewfeature/nsgflowlogs"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/clusterauthorizer"
	"github.com/Azure/go-autorest/autorest/azure"
)

type feature interface {
	Name() string
	Reconcile(ctx context.Context, instance *aropreviewv1alpha1.PreviewFeature) error
}

type Reconciler struct {
	log *logrus.Entry

	arocli        aroclient.Interface
	kubernetescli kubernetes.Interface
}

func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, kubernetescli kubernetes.Interface) *Reconciler {
	return &Reconciler{
		log:           log,
		arocli:        arocli,
		kubernetescli: kubernetescli,
	}
}

// Reconcile reconciles ARO preview features
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.PreviewV1alpha1().PreviewFeatures().Get(ctx, aropreviewv1alpha1.SingletonPreviewFeatureName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	// Get endpoints from operator
	azEnv, err := azureclient.EnvironmentFromName(instance.Spec.AZEnvironment)
	if err != nil {
		return reconcile.Result{}, err
	}

	resource, err := azure.ParseResourceID(instance.Spec.ResourceID)
	if err != nil {
		return reconcile.Result{}, err
	}

	// create refreshable authorizer from token
	authorizer, err := clusterauthorizer.NewAzRefreshableAuthorizer(ctx, r.log, &azEnv, r.kubernetescli)
	if err != nil {
		return reconcile.Result{}, err
	}

	flowLogsClient := network.NewFlowLogsClient(&azEnv, resource.SubscriptionID, authorizer)

	features := []feature{
		nsgflowlogs.NewFeature(flowLogsClient),
	}

	err = nil
	for _, f := range features {
		thisErr := f.Reconcile(ctx, instance)
		if thisErr != nil {
			// Reconcile all features even if there is an error in some of them
			err = thisErr
			r.log.Errorf("error reconciling %q: %s", f.Name(), err)
		}
	}

	// Controller-runtime will requeue when err != nil
	return reconcile.Result{}, err
}

// SetupWithManager setup our manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroPreviewFeaturePredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == aropreviewv1alpha1.SingletonPreviewFeatureName
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&aropreviewv1alpha1.PreviewFeature{}, builder.WithPredicates(aroPreviewFeaturePredicate)).
		Named(controllers.PreviewFeatureControllerName).
		Complete(r)
}
