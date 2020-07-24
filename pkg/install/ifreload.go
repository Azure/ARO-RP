package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/ifreload"
)

func (i *Installer) ensureIfReload(ctx context.Context, kubernetesClient kubernetes.Interface, securityClient securityclient.Interface) error {
	ir := ifreload.New(i.log, i.env, kubernetesClient, securityClient)
	return ir.CreateOrUpdate(ctx)
}
