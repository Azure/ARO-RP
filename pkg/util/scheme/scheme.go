package scheme

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	securityv1 "github.com/openshift/api/security/v1"
	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	appsv1defaults "k8s.io/kubernetes/pkg/apis/apps/v1"
	corev1defaults "k8s.io/kubernetes/pkg/apis/core/v1"
	rbacv1defaults "k8s.io/kubernetes/pkg/apis/rbac/v1"
	azureproviderv1beta1 "sigs.k8s.io/cluster-api-provider-azure/pkg/apis/azureprovider/v1beta1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

func init() {
	runtime.Must(apiextensions.AddToScheme(scheme.Scheme))
	runtime.Must(securityv1.AddToScheme(scheme.Scheme))
	runtime.Must(arov1alpha1.AddToScheme(scheme.Scheme))
	runtime.Must(azureproviderv1beta1.SchemeBuilder.AddToScheme(scheme.Scheme))
	runtime.Must(mcv1.AddToScheme(scheme.Scheme))
	runtime.Must(corev1.AddToScheme(scheme.Scheme))
	runtime.Must(corev1defaults.RegisterDefaults(scheme.Scheme))
	runtime.Must(appsv1.AddToScheme(scheme.Scheme))
	runtime.Must(appsv1defaults.RegisterDefaults(scheme.Scheme))
	runtime.Must(rbacv1.AddToScheme(scheme.Scheme))
	runtime.Must(rbacv1defaults.RegisterDefaults(scheme.Scheme))
	runtime.Must(machinev1beta1.AddToScheme(scheme.Scheme))
}
