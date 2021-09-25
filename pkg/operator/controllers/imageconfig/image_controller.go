package imageconfig

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/sirupsen/logrus"
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

type Reconciler struct {
	log       *logrus.Entry
	arocli    aroclient.Interface
	configcli configclient.Interface
}

func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, configcli configclient.Interface) *Reconciler {
	return &Reconciler{
		log:       log,
		arocli:    arocli,
		configcli: configcli,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	// Get cluster
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, request.Name, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	// Feature flag
	if !instance.Spec.Features.ReconcileImageConfig {
		return reconcile.Result{}, nil
	}

	// Check for cloud type
	requiredRegistries, err := getCloudAwareRegistries(instance)
	if err != nil {
		return reconcile.Result{Requeue: false}, err
	}

	// Get image.config yaml
	imageconfig, err := r.configcli.ConfigV1().Images().Get(ctx, request.Name, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	// Append to allowed registries
	if imageconfig.Spec.RegistrySources.AllowedRegistries != nil {
		imageconfig.Spec.RegistrySources.AllowedRegistries = append(imageconfig.Spec.RegistrySources.AllowedRegistries, requiredRegistries...)
	}

	// Remove from blocked registries
	if imageconfig.Spec.RegistrySources.BlockedRegistries != nil {
		removeRequiredRegistries := func(item string) bool {
			for _, v := range requiredRegistries {
				if strings.EqualFold(item, v) {
					return false
				}
			}
			return true
		}
		imageconfig.Spec.RegistrySources.BlockedRegistries = filterRegistriesInPlace(imageconfig.Spec.RegistrySources.BlockedRegistries, removeRequiredRegistries)
	}

	// Update image config registry
	_, err = r.configcli.ConfigV1().Images().Update(ctx, imageconfig, metav1.UpdateOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
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

func getCloudAwareRegistries(instance *arov1alpha1.Cluster) ([]string, error) {
	var requiredRegistries []string
	switch instance.Spec.AZEnvironment {
	case azureclient.PublicCloud.Environment.Name:
		requiredRegistries = []string{instance.Spec.ACRDomain, "arosvc." + instance.Spec.Location + ".data." + azure.PublicCloud.ContainerRegistryDNSSuffix}

	case azureclient.USGovernmentCloud.Environment.Name:
		requiredRegistries = []string{instance.Spec.ACRDomain, "arosvc." + instance.Spec.Location + ".data." + azure.USGovernmentCloud.ContainerRegistryDNSSuffix}

	default:
		err := fmt.Errorf("cloud environment %s is not supported", instance.Spec.AZEnvironment)
		return nil, err
	}
	return requiredRegistries, nil
}

func filterRegistriesInPlace(input []string, keep func(string) bool) []string {
	n := 0
	for _, x := range input {
		if keep(x) {
			input[n] = x
			n++
		}
	}
	return input[:n]
}
