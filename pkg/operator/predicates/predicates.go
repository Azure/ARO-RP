package predicates

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"strings"

	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
)

var AROCluster predicate.Predicate = predicate.NewPredicateFuncs(func(o client.Object) bool {
	return strings.EqualFold(arov1alpha1.SingletonClusterName, o.GetName())
})

var MachineRoleMaster predicate.Predicate = predicate.NewPredicateFuncs(func(o client.Object) bool {
	role, ok := o.GetLabels()["machine.openshift.io/cluster-api-machine-role"]
	return ok && strings.EqualFold("master", role)
})

var MachineRoleWorker predicate.Predicate = predicate.NewPredicateFuncs(func(o client.Object) bool {
	role, ok := o.GetLabels()["machine.openshift.io/cluster-api-machine-role"]
	return ok && strings.EqualFold("worker", role)
})

var ClusterVersion predicate.Predicate = predicate.NewPredicateFuncs(func(o client.Object) bool {
	return o.GetName() == "version"
})

var (
	pullSecretName                     = types.NamespacedName{Name: "pull-secret", Namespace: "openshift-config"}
	PullSecret     predicate.Predicate = predicate.NewPredicateFuncs(func(o client.Object) bool {
		return (o.GetName() == pullSecretName.Name && o.GetNamespace() == pullSecretName.Namespace)
	})
)

var (
	backupPullSecretName                     = types.NamespacedName{Name: operator.SecretName, Namespace: operator.Namespace}
	BackupPullSecret     predicate.Predicate = predicate.NewPredicateFuncs(func(o client.Object) bool {
		return (o.GetName() == backupPullSecretName.Name && o.GetNamespace() == backupPullSecretName.Namespace)
	})
)
