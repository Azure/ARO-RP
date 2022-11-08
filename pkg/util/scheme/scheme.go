package scheme

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	configv1 "github.com/openshift/api/config/v1"
	consolev1 "github.com/openshift/api/console/v1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	securityv1 "github.com/openshift/api/security/v1"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	mcv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	appsv1defaults "k8s.io/kubernetes/pkg/apis/apps/v1"
	corev1defaults "k8s.io/kubernetes/pkg/apis/core/v1"
	rbacv1defaults "k8s.io/kubernetes/pkg/apis/rbac/v1"

	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aropreviewv1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/preview.aro.openshift.io/v1alpha1"
)

func init() {
	utilruntime.Must(extensionsv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(securityv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(arov1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(aropreviewv1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(machinev1beta1.AddToScheme(scheme.Scheme))
	utilruntime.Must(mcv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(configv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(corev1.AddToScheme(scheme.Scheme))
	utilruntime.Must(corev1defaults.RegisterDefaults(scheme.Scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(appsv1defaults.RegisterDefaults(scheme.Scheme))
	utilruntime.Must(rbacv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(rbacv1defaults.RegisterDefaults(scheme.Scheme))
	utilruntime.Must(machinev1beta1.AddToScheme(scheme.Scheme))
	utilruntime.Must(consolev1.AddToScheme(scheme.Scheme))
	utilruntime.Must(monitoringv1.AddToScheme(scheme.Scheme))
	// AzureMachineProviderSpec is not registered by default
	scheme.Scheme.AddKnownTypes(machinev1beta1.GroupVersion, &machinev1beta1.AzureMachineProviderSpec{})
	// AzureMachineProviderSpec type has been deleted from github.com/openshift/api... manually add it.
	scheme.Scheme.AddKnownTypes(schema.GroupVersion{Group: "azureproviderconfig.openshift.io", Version: "v1beta1"}, &machinev1beta1.AzureMachineProviderSpec{})
	utilruntime.Must(hivev1.AddToScheme(scheme.Scheme))
}
