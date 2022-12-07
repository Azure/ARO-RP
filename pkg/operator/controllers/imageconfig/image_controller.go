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
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned"
	"github.com/Azure/ARO-RP/pkg/util/azureclient"
)

const (
	ControllerName = "ImageConfig"

	controllerEnabled = "aro.imageconfig.enabled"

	// Kubernetes object name
	imageConfigResource = "cluster"
)

type Reconciler struct {
	log *logrus.Entry

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

	if !instance.Spec.OperatorFlags.GetSimpleBoolean(controllerEnabled) {
		r.log.Debug("controller is disabled")
		return reconcile.Result{}, nil
	}

	r.log.Debug("running")
	requiredRegistries, err := GetCloudAwareRegistries(instance)
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
		Named(ControllerName).
		Complete(r)
}

// Switch case to ensure the correct registries are added depending on the cloud environment (Gov or Public cloud)
func GetCloudAwareRegistries(instance *arov1alpha1.Cluster) ([]string, error) {
	var replicationRegistry string
	var dnsSuffix string

	acrDomain := instance.Spec.ACRDomain
	acrSubdomain := strings.Split(acrDomain, ".")[0]
	if acrDomain == "" || acrSubdomain == "" {
		return nil, fmt.Errorf("azure container registry domain is not present or is malformed")
	}

	switch instance.Spec.AZEnvironment {
	case azureclient.PublicCloud.Environment.Name:
		dnsSuffix = azure.PublicCloud.ContainerRegistryDNSSuffix
	case azureclient.USGovernmentCloud.Environment.Name:
		dnsSuffix = azure.USGovernmentCloud.ContainerRegistryDNSSuffix
	default:
		return nil, fmt.Errorf("cloud environment %s is not supported", instance.Spec.AZEnvironment)
	}
	replicationRegistry = fmt.Sprintf("%s.%s.data.%s", acrSubdomain, instance.Spec.Location, dnsSuffix)
	return []string{acrDomain, replicationRegistry}, nil
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
