package subnets

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/predicates"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/clusterauthorizer"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

const (
	ControllerName                   = "AzureSubnets"
	controllerServiceEndpointManaged = operator.AzureSubnetsServiceEndpointManaged
)

// Reconciler is the controller struct
type Reconciler struct {
	log *logrus.Entry

	client client.Client
}

// reconcileManager is an instance of the manager instantiated per request
type reconcileManager struct {
	log *logrus.Entry

	client client.Client

	instance       *arov1alpha1.Cluster
	subscriptionID string

	subnets     subnet.Manager
	kubeSubnets subnet.KubeManager
}

// NewReconciler creates a new Reconciler
func NewReconciler(log *logrus.Entry, client client.Client) *Reconciler {
	return &Reconciler{
		log:    log,
		client: client,
	}
}

// Reconcile fixes the Network Security Groups
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance := &arov1alpha1.Cluster{}
	err := r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.AzureSubnetsEnabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running")

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(operator.AzureSubnetsNsgManaged) && !instance.Spec.OperatorFlags.GetSimpleBoolean(controllerServiceEndpointManaged) {
		// controller is disabled
		return reconcile.Result{}, nil
	}

	// Get endpoints from the operator
	azEnv, err := azureclient.EnvironmentFromName(instance.Spec.AZEnvironment)
	if err != nil {
		return reconcile.Result{}, err
	}

	resource, err := azure.ParseResourceID(instance.Spec.ResourceID)
	if err != nil {
		return reconcile.Result{}, err
	}

	// create a refreshable authorizer from token
	azRefreshAuthorizer, err := clusterauthorizer.NewAzRefreshableAuthorizer(r.log, &azEnv, r.client)
	if err != nil {
		return reconcile.Result{}, err
	}

	authorizer, err := azRefreshAuthorizer.NewRefreshableAuthorizerToken(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	manager := reconcileManager{
		log:            r.log,
		client:         r.client,
		instance:       instance,
		subscriptionID: resource.SubscriptionID,
		kubeSubnets:    subnet.NewKubeManager(r.client, resource.SubscriptionID),
		subnets:        subnet.NewManager(&azEnv, resource.SubscriptionID, authorizer),
	}

	return reconcile.Result{}, manager.reconcileSubnets(ctx)
}

func (r *reconcileManager) reconcileSubnets(ctx context.Context) error {
	subnets, err := r.kubeSubnets.List(ctx)
	if err != nil {
		return err
	}

	var combinedErrors []string

	// This potentially calls an update twice for the same loop, but this is the price
	// to pay for keeping logic split, separate, and simple
	for _, s := range subnets {
		if r.instance.Spec.OperatorFlags.GetSimpleBoolean(operator.AzureSubnetsNsgManaged) {
			err = r.ensureSubnetNSG(ctx, s)
			if err != nil {
				combinedErrors = append(combinedErrors, err.Error())
			}
		}

		if r.instance.Spec.OperatorFlags.GetSimpleBoolean(controllerServiceEndpointManaged) {
			err = r.ensureSubnetServiceEndpoints(ctx, s)
			if err != nil {
				combinedErrors = append(combinedErrors, err.Error())
			}
		}
	}

	if len(combinedErrors) > 0 {
		return fmt.Errorf("%s", strings.Join(combinedErrors, "\n"))
	}

	return nil
}

// SetupWithManager creates the controller
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arov1alpha1.Cluster{}, builder.WithPredicates(predicate.And(predicates.AROCluster, predicate.GenerationChangedPredicate{}))).
		Watches(&machinev1beta1.Machine{}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(predicates.MachineRoleMaster)).    // to reconcile on master machine replacement
		Watches(&machinev1beta1.MachineSet{}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(predicates.MachineRoleWorker)). // to reconcile on worker machines
		Named(ControllerName).
		Complete(r)
}
