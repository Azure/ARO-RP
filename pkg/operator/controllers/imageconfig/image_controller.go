package imageconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

const imageConfigResource = "cluster"

const (
	CONFIG_NAMESPACE string = "aro.imageconfig"
	ENABLED          string = CONFIG_NAMESPACE + ".enabled"
)

type Reconciler struct {
	arocli    aroclient.Interface
	configcli configclient.Interface
}

func NewReconciler(arocli aroclient.Interface, configcli configclient.Interface) *Reconciler {
	return &Reconciler{
		arocli:    arocli,
		configcli: configcli,
	}
}

// watches the ARO object for changes and reconciles image.config.openshift.io/cluster object.
// - If blockedRegistries is not nil, makes sure required registries are not added
// - If AllowedRegistries is not nil, makes sure required registries are added
// - Fails fast if both are not nil, unsupported
func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	// Get cluster
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, request.Name, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(ENABLED) {
		// controller is disabled
		return reconcile.Result{}, nil
	}

	// Check for cloud type
	requiredRegistries, err := getCloudAwareRegistries(instance)
	if err != nil {
		// Not returning error as it will requeue again
		return reconcile.Result{}, nil
	}

	// Get image.config yaml
	imageconfig, err := r.configcli.ConfigV1().Images().Get(ctx, request.Name, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	// Fail fast if both are not nil
	if imageconfig.Spec.RegistrySources.AllowedRegistries != nil && imageconfig.Spec.RegistrySources.BlockedRegistries != nil {
		err := errors.New("both AllowedRegistries and BlockedRegistries are present")
		return reconcile.Result{}, err
	}

	removeDuplicateRegistries := func(item string) bool {
		for _, v := range requiredRegistries {
			if strings.EqualFold(item, v) {
				return false
			}
		}
		return true
	}

	// Append to allowed registries
	if imageconfig.Spec.RegistrySources.AllowedRegistries != nil {

		imageconfig.Spec.RegistrySources.AllowedRegistries = filterSliceInPlace(imageconfig.Spec.RegistrySources.AllowedRegistries, removeDuplicateRegistries)
		imageconfig.Spec.RegistrySources.AllowedRegistries = append(imageconfig.Spec.RegistrySources.AllowedRegistries, requiredRegistries...)
	}

	// Remove from blocked registries
	if imageconfig.Spec.RegistrySources.BlockedRegistries != nil {
		imageconfig.Spec.RegistrySources.BlockedRegistries = filterSliceInPlace(imageconfig.Spec.RegistrySources.BlockedRegistries, removeDuplicateRegistries)
	}

	// Update image config registry
	_, err = r.configcli.ConfigV1().Images().Update(ctx, imageconfig, metav1.UpdateOptions{})
	return reconcile.Result{}, err
}

// SetupWithManager setup the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	imagePredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == imageConfigResource
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&configv1.Image{}, builder.WithPredicates(imagePredicate)).
		Named(controllers.ImageConfigControllerName).
		Complete(r)
}

// Switch case to ensure the correct registries are added depending on the cloud environment (Gov or Public cloud)
func getCloudAwareRegistries(instance *arov1alpha1.Cluster) ([]string, error) {
	var requiredRegistries []string
	switch instance.Spec.AZEnvironment {
	case azureclient.PublicCloud.Environment.Name:
		requiredRegistries = []string{instance.Spec.ACRDomain, "arosvc." + instance.Spec.Location + ".data." + azure.PublicCloud.ContainerRegistryDNSSuffix}

	case azureclient.USGovernmentCloud.Environment.Name:
		requiredRegistries = []string{instance.Spec.ACRDomain, "arosvc." + instance.Spec.Location + ".data." + azure.USGovernmentCloud.ContainerRegistryDNSSuffix}

	default:
		return nil, fmt.Errorf("cloud environment %s is not supported", instance.Spec.AZEnvironment)
	}
	return requiredRegistries, nil
}

// Helper function that filters registries to make sure they are added in consistent order
func filterSliceInPlace(input []string, keep func(string) bool) []string {
	n := 0
	for _, x := range input {
		if keep(x) {
			input[n] = x
			n++
		}
	}
	return input[:n]
}
