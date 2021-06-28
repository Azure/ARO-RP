package azurensg

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
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/clusterauthorizer"
)

// AzureNSGReconciler is the controller struct
type AzureNSGReconciler struct {
	arocli        aroclient.Interface
	maocli        maoclient.Interface
	kubernetescli kubernetes.Interface
	log           *logrus.Entry
}

// NewReconciler creates a new Reconciler
func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, maocli maoclient.Interface, kubernetescli kubernetes.Interface) *AzureNSGReconciler {
	return &AzureNSGReconciler{
		arocli:        arocli,
		maocli:        maocli,
		kubernetescli: kubernetescli,
		log:           log,
	}
}

//Reconcile fixes the Network Security Groups
func (r *AzureNSGReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.Features.ReconcileNSGs {
		// reconciling NSGs is disabled
		return reconcile.Result{}, nil
	}

	// Get endpoints from operator
	azEnv, err := azureclient.EnvironmentFromName(instance.Spec.AZEnvironment)
	if err != nil {
		return reconcile.Result{}, err
	}
	// Grab azure-credentials from secret
	credentials, err := clusterauthorizer.AzCredentials(ctx, r.kubernetescli)
	if err != nil {
		return reconcile.Result{}, err
	}
	resource, err := azure.ParseResourceID(instance.Spec.ResourceID)
	if err != nil {
		return reconcile.Result{}, err
	}
	// create service principal token from azure-credentials
	token, err := aad.GetToken(ctx, r.log, string(credentials.ClientID), string(credentials.ClientSecret), string(credentials.TenantID), azEnv.ActiveDirectoryEndpoint, azEnv.ResourceManagerEndpoint)
	if err != nil {
		return reconcile.Result{}, err
	}
	// create refreshable authorizer from token
	authorizer, err := clusterauthorizer.NewAzRefreshableAuthorizer(token)
	if err != nil {
		return reconcile.Result{}, err
	}
	subnetsClient := network.NewSubnetsClient(&azEnv, resource.SubscriptionID, authorizer)

	return reconcile.Result{}, r.reconcileSubnetNSG(ctx, instance, resource.SubscriptionID, subnetsClient)
}

// SetupWithManager creates the controller
func (r *AzureNSGReconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Watches(&source.Kind{Type: &machinev1beta1.Machine{}}, &handler.EnqueueRequestForObject{}). // to reconcile on machine replacement
		Watches(&source.Kind{Type: &corev1.Node{}}, &handler.EnqueueRequestForObject{}).            // to reconcile on node status change
		Named(controllers.AzureNSGControllerName).
		Complete(r)
}
