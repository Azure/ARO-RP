package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	operatorclient "github.com/openshift/client-go/operator/clientset/versioned"
	securityclient "github.com/openshift/client-go/security/clientset/versioned"
	samplesclient "github.com/openshift/cluster-samples-operator/pkg/generated/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
)

type Interface interface {
	InitializeClients(ctx context.Context) error

	Get(ctx context.Context, groupKind, namespace, name string) ([]byte, error)
	List(ctx context.Context, groupKind, namespace string) ([]byte, error)
	CreateOrUpdate(ctx context.Context, obj *unstructured.Unstructured) error
	Delete(ctx context.Context, groupKind, namespace, name string) error

	ClusterUpgrade(ctx context.Context) error
	MustGather(ctx context.Context, w io.Writer) error
	EnsureGenevaLogging(ctx context.Context) error
	DisableAlertManagerWarning(ctx context.Context) error
	DisableUpdates(ctx context.Context) error
	DisableSamples(ctx context.Context) error
	DisableOperatorHubSources(ctx context.Context) error
	UpdateConsoleBranding(ctx context.Context) error

	GatherFailureLogs(ctx context.Context)

	APIServersReady() (bool, error)
	OperatorConsoleExists() (bool, error)
	OperatorConsoleReady() (bool, error)
	ClusterVersionReady() (bool, error)
	BootstrapConfigMapReady() (bool, error)
	IngressControllerReady() (bool, error)
}

type adminactions struct {
	log *logrus.Entry
	env env.Interface
	oc  *api.OpenShiftCluster

	cli          kubernetes.Interface
	discoverycli *discovery.DiscoveryClient
	dynamiccli   dynamic.Interface
	configcli    configclient.Interface
	operatorcli  operatorclient.Interface
	seccli       securityclient.Interface
	samplescli   samplesclient.Interface
}

func New(log *logrus.Entry, env_ env.Interface, oc *api.OpenShiftCluster) Interface {
	return &adminactions{
		log: log,
		env: env_,
		oc:  oc,
	}
}

func (a *adminactions) Get(ctx context.Context, groupKind, namespace, name string) ([]byte, error) {
	_, apiresources, err := a.discoverycli.ServerGroupsAndResources()
	if err != nil {
		return nil, err
	}

	gvr, err := a.findGVR(apiresources, groupKind, "")
	if err != nil {
		return nil, err
	}

	un, err := a.dynamiccli.Resource(*gvr).Namespace(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return un.MarshalJSON()
}

func (a *adminactions) List(ctx context.Context, groupKind, namespace string) ([]byte, error) {
	_, apiresources, err := a.discoverycli.ServerGroupsAndResources()
	if err != nil {
		return nil, err
	}

	gvr, err := a.findGVR(apiresources, groupKind, "")
	if err != nil {
		return nil, err
	}

	ul, err := a.dynamiccli.Resource(*gvr).Namespace(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return ul.MarshalJSON()
}

func (a *adminactions) CreateOrUpdate(ctx context.Context, obj *unstructured.Unstructured) error {
	// TODO log changes

	namespace := obj.GetNamespace()
	groupKind := obj.GroupVersionKind().GroupKind().String()

	_, apiresources, err := a.discoverycli.ServerGroupsAndResources()
	if err != nil {
		return err
	}

	gvr, err := a.findGVR(apiresources, groupKind, "")
	if err != nil {
		return err
	}

	_, err = a.dynamiccli.Resource(*gvr).Namespace(namespace).Update(obj, metav1.UpdateOptions{})
	if !errors.IsNotFound(err) {
		return err
	}

	_, err = a.dynamiccli.Resource(*gvr).Namespace(namespace).Create(obj, metav1.CreateOptions{})
	return err
}

func (a *adminactions) Delete(ctx context.Context, groupKind, namespace, name string) error {
	// TODO log changes

	_, apiresources, err := a.discoverycli.ServerGroupsAndResources()
	if err != nil {
		return err
	}

	gvr, err := a.findGVR(apiresources, groupKind, "")
	if err != nil {
		return err
	}

	return a.dynamiccli.Resource(*gvr).Namespace(namespace).Delete(name, &metav1.DeleteOptions{})
}
