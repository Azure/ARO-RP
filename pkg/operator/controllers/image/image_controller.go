package image

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	// arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/operator/controllers"
	"github.com/Azure/go-autorest/autorest/azure"
	configv1 "github.com/openshift/api/config/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/sirupsen/logrus"
	"github.com/ugorji/go/codec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const imageConfigResource = "cluster"

// var allowedRegistries = []string{"arosvc.azurecr.io", "quay.io"}
var allowedRegistries = []string{}

type Reconciler struct {
	arocli     aroclient.Interface
	configcli  configclient.Interface
	log        *logrus.Entry
	jsonHandle *codec.JsonHandle
}

func NewReconciler(log *logrus.Entry, arocli aroclient.Interface, configcli configclient.Interface) *Reconciler {
	return &Reconciler{
		arocli:     arocli,
		log:        log,
		jsonHandle: new(codec.JsonHandle),
		configcli:  configcli,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	instance, err := r.arocli.AroV1alpha1().Clusters().Get(ctx, request.Name, metav1.GetOptions{})

	if err != nil {
		return reconcile.Result{}, err
	}

	// ! Feature flag code
	// if !instance.Spec.Features.ReconcileImageConfig {
	// 	return reconcile.Result{}, nil
	// }

	if instance.Spec.AZEnvironment == "AzurePublicCloud" {
		allowedRegistries = []string{"arosvc.azurecr.io", "arosvc." + instance.Spec.Location + ".data." + azure.PublicCloud.ContainerRegistryDNSSuffix}
	}

	// * 1. Get image.config yaml
	imageconfig, err := r.configcli.ConfigV1().Images().Get(ctx, request.Name, metav1.GetOptions{})
	if err != nil {
		return reconcile.Result{}, err
	}

	// * 2. get allowedRegistryMap, one of them has to be nil as they are mutually exlusive
	allowedRegistryMap := imageconfig.Spec.RegistrySources.AllowedRegistries
	// blockedRegistryMap := imageconfig.Spec.RegistrySources.BlockedRegistries

	var regMap = make(map[string]bool)
	// * 3. case allowedRegistry map is not nil
	if allowedRegistryMap != nil {
		// * 4. Set all registries to false by default
		for _, registry := range allowedRegistries {
			regMap[registry] = false
		}
		// * 5. Set only those registries to true that exist in image.config
		for _, allowedRegistry := range allowedRegistryMap {
			if _, ok := regMap[allowedRegistry]; ok {
				regMap[allowedRegistry] = true
			}
		}
		// * 6. for registries that don't exist image.config, add to image.config
		for registryName := range regMap {
			if !regMap[registryName] {
				imageconfig.Spec.RegistrySources.AllowedRegistries = append(imageconfig.Spec.RegistrySources.AllowedRegistries, registryName)
				_, err := r.configcli.ConfigV1().Images().Update(ctx, imageconfig, metav1.UpdateOptions{})
				if err != nil {
					return reconcile.Result{}, err
				}
			}
		}
	}

	return reconcile.Result{}, nil
}

// SetupWithManager setup the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.log.Info("Starting image controller")

	imagePredicate := predicate.NewPredicateFuncs(func(o client.Object) bool {
		return o.GetName() == imageConfigResource
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&configv1.Image{}, builder.WithPredicates(imagePredicate)).
		Named(controllers.ImageConfigControllerName).
		Complete(r)
}
