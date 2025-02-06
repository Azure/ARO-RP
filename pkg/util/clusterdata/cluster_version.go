package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	machineclient "github.com/openshift/client-go/machine/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"

	"github.com/Azure/ARO-RP/pkg/api"
)

type clusterVersionEnricher struct {
}

func (ce clusterVersionEnricher) Enrich(
	ctx context.Context,
	log *logrus.Entry,
	oc *api.OpenShiftCluster,
	k8scli kubernetes.Interface,
	configcli configclient.Interface,
	machinecli machineclient.Interface,
	operatorcli operatorclient.Interface,
) error {
	cv, err := configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return err
	}

	oc.Lock.Lock()
	defer oc.Lock.Unlock()

	oc.Properties.ClusterProfile.Version = cv.Status.Desired.Version
	return nil
}

func (ce clusterVersionEnricher) SetDefaults(oc *api.OpenShiftCluster) {
	oc.Properties.ClusterProfile.Version = ""
}
