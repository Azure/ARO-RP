package storageaccounts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	imageregistryclient "github.com/openshift/client-go/imageregistry/clientset/versioned"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
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
	"github.com/Azure/ARO-RP/pkg/util/aad"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
	"github.com/Azure/ARO-RP/pkg/util/clusterauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

const (
	ControllerName = "StorageAccounts"

	controllerEnabled = "aro.storageaccounts.enabled"
)

// Reconciler is the controller struct
type Reconciler struct {
	log *logrus.Entry

	arocli           aroclient.Interface
	kubernetescli    kubernetes.Interface
	maocli           machineclient.Interface
	imageregistrycli imageregistryclient.Interface
}

// reconcileManager is instance of manager instantiated per request
type reconcileManager struct {
	log *logrus.Entry

	instance       *arov1alpha1.Cluster
	subscriptionID string

	imageregistrycli imageregistryclient.Interface
	kubeSubnets      subnet.KubeManager
	storage          storage.AccountsClient
}

// NewReconciler creates a new Reconciler
func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, maocli machineclient.Interface, kubernetescli kubernetes.Interface, imageregistrycli imageregistryclient.Interface) *Reconciler {
	return &Reconciler{
		log:              log,
		arocli:           arocli,
		kubernetescli:    kubernetescli,
		imageregistrycli: imageregistrycli,
		maocli:           maocli,
	}
}

// Reconcile ensures the firewall is set on storage accounts as per user subnets
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, arov1alpha1.SingletonClusterName, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(controllerEnabled) {
		// controller is disabled
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
	azRefreshAuthorizer, err := clusterauthorizer.NewAzRefreshableAuthorizer(r.log, &azEnv, r.kubernetescli, aad.NewTokenClient())
	if err != nil {
		return reconcile.Result{}, err
	}

	authorizer, err := azRefreshAuthorizer.NewRefreshableAuthorizerToken(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	manager := reconcileManager{
		log:            r.log,
		instance:       instance,
		subscriptionID: resource.SubscriptionID,

		imageregistrycli: r.imageregistrycli,
		kubeSubnets:      subnet.NewKubeManager(r.maocli, resource.SubscriptionID),
		storage:          storage.NewAccountsClient(&azEnv, resource.SubscriptionID, authorizer),
	}

	return reconcile.Result{}, manager.reconcileAccounts(ctx)
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
		Named(ControllerName).
		Complete(r)
}
