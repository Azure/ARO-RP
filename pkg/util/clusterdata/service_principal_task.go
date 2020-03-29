package clusterdata

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/api"
)

const (
	cloudProviderConfigNamespace = "openshift-config"
	cloudProviderConfigName      = "cloud-provider-config"
)

func newServicePrincipalEnricherTask(log *logrus.Entry, restConfig *rest.Config, oc *api.OpenShiftCluster) (enricherTask, error) {
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &servicePrincipalEnricherTask{
		log:    log,
		client: client,
		oc:     oc,
	}, nil
}

type servicePrincipalEnricherTask struct {
	log    *logrus.Entry
	client kubernetes.Interface
	oc     *api.OpenShiftCluster
}

func (ef *servicePrincipalEnricherTask) FetchData(callbacks chan<- func(), errs chan<- error) {
	cm, err := ef.client.CoreV1().ConfigMaps(cloudProviderConfigNamespace).Get(cloudProviderConfigName, metav1.GetOptions{})
	if err != nil {
		ef.log.Error(err)
		errs <- err
		return
	}

	var config cloudProviderConfig
	if err := json.Unmarshal([]byte(cm.Data["config"]), &config); err != nil {
		ef.log.Error(err)
		errs <- err
		return
	}

	callbacks <- func() {
		ef.oc.Properties.ServicePrincipalProfile.TenantID = config.TenantID
		ef.oc.Properties.ServicePrincipalProfile.ClientID = config.AADClientID
	}
}

func (ef *servicePrincipalEnricherTask) SetDefaults() {
	ef.oc.Properties.ServicePrincipalProfile.TenantID = ""
	ef.oc.Properties.ServicePrincipalProfile.ClientID = ""
}
