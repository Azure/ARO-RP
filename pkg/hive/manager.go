package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	hiveclient "github.com/openshift/hive/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Interface interface {
}

type client struct {
	hiveClientset *hiveclient.Clientset
	kubernetescli *kubernetes.Clientset
}

func New(hiveClientset *hiveclient.Clientset, kubernetescli *kubernetes.Clientset) Interface {
	return &client{
		hiveClientset: hiveClientset,
		kubernetescli: kubernetescli,
	}
}

func NewFromConfig(restConfig *rest.Config) (Interface, error) {
	hiveclientset, err := hiveclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	hiveKubernetescli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return New(hiveclientset, hiveKubernetescli), nil
}
