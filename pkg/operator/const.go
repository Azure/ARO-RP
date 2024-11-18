package operator

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

const (
	RoleMaster = "master"
	RoleWorker = "worker"

	Namespace  = "openshift-azure-operator"
	SecretName = "cluster"

	OperatorIdentityName       = "aro-operator"
	OperatorIdentitySecretName = "azure-cloud-credentials"
	OperatorTokenFile          = "/var/run/secrets/openshift/serviceaccount/token"
)
