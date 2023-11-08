package storageaccounts

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	imageregistryv1 "github.com/openshift/api/imageregistry/v1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/clusterauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/stringutils"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

const (
	ControllerName = "StorageAccounts"

	controllerEnabled = "aro.storageaccounts.enabled"
)

// Reconciler is the controller struct
type Reconciler struct {
	log *logrus.Entry

	client client.Client

	newManager newManager
}

// NewReconciler creates a new Reconciler
func NewReconciler(log *logrus.Entry, client client.Client) *Reconciler {
	return &Reconciler{
		log:        log,
		client:     client,
		newManager: newReconcileManager,
	}
}

// Reconcile ensures the firewall is set on storage accounts as per user subnets
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance := &arov1alpha1.Cluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(controllerEnabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running")

	location := instance.Spec.Location
	resource, err := azure.ParseResourceID(instance.Spec.ResourceID)
	if err != nil {
		return reconcile.Result{}, err
	}
	subscriptionId := resource.SubscriptionID
	managedResourceGroupName := stringutils.LastTokenByte(instance.Spec.ClusterResourceGroupID, '/')

	azEnv, authorizer, err := r.getAzureAuthorizer(ctx, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	manager := r.newManager(
		r.log,
		location, subscriptionId, managedResourceGroupName,
		azEnv, authorizer,
	)

	subnets, err := r.getSubnetsToReconcile(ctx, instance, subscriptionId, manager)
	if err != nil {
		return reconcile.Result{}, err
	}

	storageAccounts, err := r.getStorageAccountNames(ctx, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = manager.reconcileAccounts(ctx, subnets, storageAccounts)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) getAzureAuthorizer(ctx context.Context, instance *arov1alpha1.Cluster) (azureclient.AROEnvironment, autorest.Authorizer, error) {
	// Get endpoints from operator
	azEnv, err := azureclient.EnvironmentFromName(instance.Spec.AZEnvironment)
	if err != nil {
		return azureclient.AROEnvironment{}, nil, err
	}

	// create refreshable authorizer from token
	azRefreshAuthorizer, err := clusterauthorizer.NewAzRefreshableAuthorizer(r.log, &azEnv, r.client)
	if err != nil {
		return azureclient.AROEnvironment{}, nil, err
	}

	authorizer, err := azRefreshAuthorizer.NewRefreshableAuthorizerToken(ctx)
	if err != nil {
		return azureclient.AROEnvironment{}, nil, err
	}

	return azEnv, authorizer, nil
}

func (r *Reconciler) getSubnetsToReconcile(ctx context.Context, instance *arov1alpha1.Cluster, subscriptionId string, m manager) ([]string, error) {
	subnets := []string{}
	subnets = append(subnets, instance.Spec.ServiceSubnets...)

	clusterSubnets, err := r.getClusterSubnets(ctx, subscriptionId)
	if err != nil {
		return nil, err
	}
	clusterSubnetsToReconcile, err := m.checkClusterSubnetsToReconcile(ctx, clusterSubnets)
	if err != nil {
		return nil, err
	}
	subnets = append(subnets, clusterSubnetsToReconcile...)

	return subnets, nil
}

func (r *Reconciler) getClusterSubnets(ctx context.Context, subscriptionId string) ([]string, error) {
	kubeManager := subnet.NewKubeManager(r.client, subscriptionId)

	subnets := []string{}

	clusterSubnets, err := kubeManager.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, subnet := range clusterSubnets {
		subnets = append(subnets, subnet.ResourceID)
	}
	return subnets, nil
}

func (r *Reconciler) getStorageAccountNames(ctx context.Context, instance *arov1alpha1.Cluster) ([]string, error) {
	rc := &imageregistryv1.Config{}
	err := r.client.Get(ctx, types.NamespacedName{Name: "cluster"}, rc)
	if err != nil {
		return nil, err
	}
	if rc.Spec.Storage.Azure == nil {
		return nil, fmt.Errorf("azure storage field is nil in image registry config")
	}

	return []string{
		"cluster" + instance.Spec.StorageSuffix, // this is our creation, so name is deterministic
		rc.Spec.Storage.Azure.AccountName,
	}, nil
}

// SetupWithManager creates the controller
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	aroClusterPredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == arov1alpha1.SingletonClusterName
	})
	masterMachinePredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		role, ok := o.GetLabels()["machine.openshift.io/cluster-api-machine-role"]
		return ok && role == "master"
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(aroClusterPredicate)).
		Watches(&source.Kind{Type: &machinev1beta1.Machine{}}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(masterMachinePredicate)). // to reconcile on master machine replacement
		Watches(&source.Kind{Type: &machinev1beta1.MachineSet{}}, &handler.EnqueueRequestForObject{}).                                              // to reconcile on worker machinesets
		Named(ControllerName).
		Complete(r)
}
