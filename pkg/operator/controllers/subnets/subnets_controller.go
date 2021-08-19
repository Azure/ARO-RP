package subnets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure"
	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maoclient "github.com/openshift/machine-api-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/clusterauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

// Reconciler is the controller struct
type Reconciler struct {
	log *logrus.Entry

	arocli        aroclient.Interface
	kubernetescli kubernetes.Interface
	maocli        maoclient.Interface
}

// reconcileManager is instance of manager instanciated per request
type reconcileManager struct {
	log *logrus.Entry

	instance       *arov1alpha1.Cluster
	subscriptionID string

	subnets     subnet.Manager
	kubeSubnets subnet.KubeManager
}

// NewReconciler creates a new Reconciler
func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, kubernetescli kubernetes.Interface, maocli maoclient.Interface) *Reconciler {
	return &Reconciler{
		log:           log,
		arocli:        arocli,
		kubernetescli: kubernetescli,
		maocli:        maocli,
	}
}

//Reconcile fixes the Network Security Groups
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.Features.ReconcileSubnets {
		// reconciling subnets is disabled
		return reconcile.Result{}, nil
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

	manager := reconcileManager{
		log:            r.log,
		instance:       instance,
		subscriptionID: resource.SubscriptionID,
		kubeSubnets:    subnet.NewKubeManager(r.maocli, resource.SubscriptionID),
		subnets:        subnet.NewManager(&azEnv, resource.SubscriptionID, authorizer),
	}

	return reconcile.Result{}, manager.reconcileSubnets(ctx)
}

// SetupWithManager creates the controller
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Watches(&source.Kind{Type: &machinev1beta1.Machine{}}, &handler.EnqueueRequestForObject{}). // to reconcile on machine replacement
		Watches(&source.Kind{Type: &corev1.Node{}}, &handler.EnqueueRequestForObject{}).            // to reconcile on node status change
		Named(controllers.AzureSubnetsControllerName).
		Complete(r)
}
