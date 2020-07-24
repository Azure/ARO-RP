package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/genevalogging"
)

func (i *Installer) ensureGenevaLogging(ctx context.Context, kubernetesClient kubernetes.Interface, securityClient securityclient.Interface) error {
	gl := genevalogging.New(i.log, i.env, i.doc.OpenShiftCluster, kubernetesClient, securityClient)
	return gl.CreateOrUpdate(ctx)
}
