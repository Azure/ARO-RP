package previewfeature

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest/azure"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aropreviewv1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/preview.aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers/previewfeature/nsgflowlogs"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/armnetwork"
	"github.com/Azure/ARO-RP/pkg/util/subnet"
)

const (
	ControllerName = "PreviewFeature"
)

type feature interface {
	Name() string
	Reconcile(ctx context.Context, instance *aropreviewv1alpha1.PreviewFeature) error
}

type Reconciler struct {
	log *logrus.Entry

	client client.Client
}

func NewReconciler(log *logrus.Entry, client client.Client) *Reconciler {
	return &Reconciler{
		log:    log,
		client: client,
	}
}

// Reconcile reconciles ARO preview features
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	r.log.Debug("running")
	instance := &aropreviewv1alpha1.PreviewFeature{}
	err := r.client.Get(ctx, types.NamespacedName{Name: aropreviewv1alpha1.SingletonPreviewFeatureName}, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	clusterInstance := &arov1alpha1.Cluster{}
	err = r.client.Get(ctx, types.NamespacedName{Name: arov1alpha1.SingletonClusterName}, clusterInstance)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Get endpoints from operator
	azEnv, err := azureclient.EnvironmentFromName(clusterInstance.Spec.AZEnvironment)
	if err != nil {
		return reconcile.Result{}, err
	}

	resource, err := azure.ParseResourceID(clusterInstance.Spec.ResourceID)
	if err != nil {
		return reconcile.Result{}, err
	}

	credential, err := azidentity.NewDefaultAzureCredential(azEnv.DefaultAzureCredentialOptions())
	if err != nil {
		return reconcile.Result{}, err
	}

	options := azEnv.ArmClientOptions()

	flowLogsClient, err := armnetwork.NewFlowLogsClient(resource.SubscriptionID, credential, options)
	if err != nil {
		return reconcile.Result{}, err
	}

	kubeSubnets := subnet.NewKubeManager(r.client, resource.SubscriptionID)
	subnets, err := armnetwork.NewSubnetsClient(resource.SubscriptionID, credential, options)
	if err != nil {
		return reconcile.Result{}, err
	}

	features := []feature{
		nsgflowlogs.NewFeature(flowLogsClient, kubeSubnets, subnets, clusterInstance.Spec.Location),
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
		Named(ControllerName).
		Complete(r)
}
