package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
)

func newClusterServicePrincipalEnricherTask(log *logrus.Entry, restConfig *rest.Config, oc *api.OpenShiftCluster) (enricherTask, error) {
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &clusterServicePrincipalEnricherTask{
		log:    log,
		client: client,
		oc:     oc,
	}, nil
}

type clusterServicePrincipalEnricherTask struct {
	log    *logrus.Entry
	client kubernetes.Interface
	oc     *api.OpenShiftCluster
}

func (ef *clusterServicePrincipalEnricherTask) FetchData(ctx context.Context, callbacks chan<- func(), errs chan<- error) {
	secret, err := ef.client.CoreV1().Secrets("kube-system").Get(ctx, "azure-credentials", metav1.GetOptions{})
	if err != nil {
		ef.log.Error(err)
		errs <- err
		return
	}

	callbacks <- func() {
		ef.oc.Properties.ServicePrincipalProfile.ClientID = string(secret.Data["azure_client_id"])
		ef.oc.Properties.ServicePrincipalProfile.ClientSecret = api.SecureString(secret.Data["azure_client_secret"])
	}
}

func (ef *clusterServicePrincipalEnricherTask) SetDefaults() {}
