package kubeactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

type Interface interface {
	Get(oc *api.OpenShiftCluster, groupKind, namespace, name string) ([]byte, error)
	List(oc *api.OpenShiftCluster, groupKind, namespace string) ([]byte, error)
	CreateOrUpdate(oc *api.OpenShiftCluster, obj *unstructured.Unstructured) error
	Delete(oc *api.OpenShiftCluster, groupKind, namespace, name string) error
	MustGather(ctx context.Context, oc *api.OpenShiftCluster, w io.Writer) error
}

type kubeactions struct {
	log *logrus.Entry
	env env.Interface
}

func New(log *logrus.Entry, env env.Interface) Interface {
	return &kubeactions{
		log: log,
		env: env,
	}
}

func (ka *kubeactions) Get(oc *api.OpenShiftCluster, groupKind, namespace, name string) ([]byte, error) {
	restConfig, err := restconfig.RestConfig(ka.env, oc)
	if err != nil {
		return nil, err
	}
	dh, err := dynamichelper.New(ka.log, restConfig, dynamichelper.UpdatePolicy{})
	if err != nil {
		return nil, err
	}

	un, err := dh.Get(groupKind, namespace, name)
	if err != nil {
		return nil, err
	}

	return un.MarshalJSON()
}

func (ka *kubeactions) List(oc *api.OpenShiftCluster, groupKind, namespace string) ([]byte, error) {
	restConfig, err := restconfig.RestConfig(ka.env, oc)
	if err != nil {
		return nil, err
	}
	dh, err := dynamichelper.New(ka.log, restConfig, dynamichelper.UpdatePolicy{})
	if err != nil {
		return nil, err
	}

	ul, err := dh.List(groupKind, namespace)
	if err != nil {
		return nil, err
	}

	return ul.MarshalJSON()
}

func (ka *kubeactions) CreateOrUpdate(oc *api.OpenShiftCluster, obj *unstructured.Unstructured) error {
	// TODO log changes

	restConfig, err := restconfig.RestConfig(ka.env, oc)
	if err != nil {
		return err
	}
	dh, err := dynamichelper.New(ka.log, restConfig, dynamichelper.UpdatePolicy{})
	if err != nil {
		return err
	}

	return dh.CreateOrUpdate(obj)
}

func (ka *kubeactions) Delete(oc *api.OpenShiftCluster, groupKind, namespace, name string) error {
	// TODO log changes

	restConfig, err := restconfig.RestConfig(ka.env, oc)
	if err != nil {
		return err
	}
	dh, err := dynamichelper.New(ka.log, restConfig, dynamichelper.UpdatePolicy{})
	if err != nil {
		return err
	}

	return dh.Delete(groupKind, namespace, name)
}
