package storageaccounts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest/azure"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/storage"
	"github.com/Azure/ARO-RP/pkg/util/clusterauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

const (
	ControllerName = "StorageAccounts"
)

// Reconciler is the controller struct
type Reconciler struct {
	log *logrus.Entry

	client client.Client
}

// reconcileManager is instance of manager instantiated per request
type reconcileManager struct {
	log *logrus.Entry

	instance       *arov1alpha1.Cluster
	subscriptionID string

	client      client.Client
	kubeSubnets subnet.KubeManager
	subnets     armnetwork.SubnetsClient
	storage     storage.AccountsClient
}

// NewReconciler creates a new Reconciler
func NewReconciler(log *logrus.Entry, client client.Client) *Reconciler {
	return &Reconciler{
		log:    log,
		client: client,
	}
}

// Reconcile ensures the firewall is set on storage accounts as per user subnets
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance := &arov1alpha1.Cluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.StorageAccountsEnabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running")

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
	azRefreshAuthorizer, err := clusterauthorizer.NewAzRefreshableAuthorizer(r.log, &azEnv, r.client)
	if err != nil {
		return reconcile.Result{}, err
	}

	authorizer, err := azRefreshAuthorizer.NewRefreshableAuthorizerToken(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	tokenCredential, err := azidentity.NewDefaultAzureCredential(azEnv.DefaultAzureCredentialOptions())
	if err != nil {
		return reconcile.Result{}, err
	}

	clientOptions := &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: azEnv.Cloud,
		},
	}

	subnetsClient, err := armnetwork.NewSubnetsClient(resource.SubscriptionID, tokenCredential, clientOptions)
	if err != nil {
		return reconcile.Result{}, err
	}

	manager := reconcileManager{
		log:            r.log,
		instance:       instance,
		subscriptionID: resource.SubscriptionID,

		client:      r.client,
		kubeSubnets: subnet.NewKubeManager(r.client, resource.SubscriptionID),
		subnets:     subnetsClient,
		storage:     storage.NewAccountsClient(&azEnv, resource.SubscriptionID, authorizer),
	}

	return reconcile.Result{}, manager.reconcileAccounts(ctx)
}

// SetupWithManager creates the controller
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		Watches(&source.Kind{Type: &machinev1beta1.Machine{}}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(predicates.MachineRoleMaster)).    // to reconcile on master machine replacement
		Watches(&source.Kind{Type: &machinev1beta1.MachineSet{}}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(predicates.MachineRoleWorker)). // to reconcile on worker machines
		Named(ControllerName).
		Complete(r)
}
