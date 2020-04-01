package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
)

func newClusterVersionEnricherTask(log *logrus.Entry, restConfig *rest.Config, oc *api.OpenShiftCluster) (enricherTask, error) {
	client, err := configclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &clusterVersionEnricherTask{
		log:    log,
		client: client,
		oc:     oc,
	}, nil
}

type clusterVersionEnricherTask struct {
	log    *logrus.Entry
	client configclient.Interface
	oc     *api.OpenShiftCluster
}

func (ef *clusterVersionEnricherTask) FetchData(callbacks chan<- func(), errs chan<- error) {
	cv, err := ef.client.ConfigV1().ClusterVersions().Get("version", metav1.GetOptions{})
	if err != nil {
		ef.log.Error(err)
		errs <- err
		return
	}

	callbacks <- func() {
		ef.oc.Properties.ClusterProfile.Version = cv.Status.Desired.Version
	}
}

func (ef *clusterVersionEnricherTask) SetDefaults() {
	ef.oc.Properties.ClusterProfile.Version = ""
}
