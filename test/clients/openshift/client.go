package openshift

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/discovery"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	authorizationv1client "k8s.io/client-go/kubernetes/typed/authorization/v1"
	batchv1client "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	policyv1beta1client "k8s.io/client-go/kubernetes/typed/policy/v1beta1"
	rbacv1client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"
	v1 "k8s.io/client-go/tools/clientcmd/api/v1"
)

type Client struct {
	config          *rest.Config
	Discovery       discovery.DiscoveryInterface
	AppsV1          appsv1client.AppsV1Interface
	AuthorizationV1 authorizationv1client.AuthorizationV1Interface
	CoreV1          corev1client.CoreV1Interface
	BatchV1         batchv1client.BatchV1Interface
	PolicyV1beta1   policyv1beta1client.PolicyV1beta1Interface
	RbacV1          rbacv1client.RbacV1Interface
}

func newClientFromRestConfig(config *rest.Config) *Client {
	return &Client{
		config:          config,
		Discovery:       discovery.NewDiscoveryClientForConfigOrDie(config),
		AppsV1:          appsv1client.NewForConfigOrDie(config),
		AuthorizationV1: authorizationv1client.NewForConfigOrDie(config),
		CoreV1:          corev1client.NewForConfigOrDie(config),
		PolicyV1beta1:   policyv1beta1client.NewForConfigOrDie(config),
		RbacV1:          rbacv1client.NewForConfigOrDie(config),
		BatchV1:         batchv1client.NewForConfigOrDie(config),
	}
}

func restConfigFromV1Config(kc *v1.Config) (*rest.Config, error) {
	var c kapi.Config
	err := latest.Scheme.Convert(kc, &c, nil)
	if err != nil {
		return nil, err
	}

	kubeconfig := clientcmd.NewDefaultClientConfig(c, &clientcmd.ConfigOverrides{})
	return kubeconfig.ClientConfig()
}

func NewAdminClient(log *logrus.Entry, config *v1.Config) (*Client, error) {
	restconfig, err := restConfigFromV1Config(config)
	if err != nil {
		return nil, err
	}

	return newClientFromRestConfig(restconfig), nil
}
