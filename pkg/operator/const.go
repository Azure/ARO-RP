package operator

import "k8s.io/apimachinery/pkg/types"

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	RoleMaster = "master"
	RoleWorker = "worker"

	Namespace  = "openshift-azure-operator"
	SecretName = "cluster"
)

var SecretKey = types.NamespacedName{
	Name:      SecretName,
	Namespace: Namespace,
}

var ControlPlaneDeployment = types.NamespacedName{
	Name:      "aro-operator-master",
	Namespace: Namespace,
}
var WorkerDeployment = types.NamespacedName{
	Name:      "aro-operator-worker",
	Namespace: Namespace,
}
