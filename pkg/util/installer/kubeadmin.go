package installer

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
)

// See github.com/openshift/installer/pkg/asset/password
type KubeadminPasswordData struct {
	Password string
}

type AdminKubeConfigSignerCertKey struct {
	SelfSignedCertKey
}

type kubeconfig struct {
	Config *clientcmdv1.Config
}

// AdminInternalClient is the asset for the admin kubeconfig.
type AdminInternalClient struct {
	kubeconfig
}
