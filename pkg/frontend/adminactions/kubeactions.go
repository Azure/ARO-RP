package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configclient "github.com/openshift/client-go/config/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/azureclient/mgmt/network"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/restconfig"
)

// KubeActions are those that involve k8s objects, and thus depend upon k8s clients being createable
type KubeActions interface {
	KubeGet(ctx context.Context, groupKind, namespace, name string) ([]byte, error)
	KubeList(ctx context.Context, groupKind, namespace string) ([]byte, error)
	KubeCreateOrUpdate(ctx context.Context, obj *unstructured.Unstructured) error
	KubeDelete(ctx context.Context, groupKind, namespace, name string) error
	Upgrade(ctx context.Context, upgradeY bool) error
}

type kubeActions struct {
	log       *logrus.Entry
	oc        *api.OpenShiftCluster
	dh        dynamichelper.Interface
	configcli configclient.Interface

	virtualNetworks network.VirtualNetworksClient
}

// NewKubeActions returns a kubeActions
func NewKubeActions(log *logrus.Entry, env env.Interface, oc *api.OpenShiftCluster,
	subscriptionDoc *api.SubscriptionDocument) (KubeActions, error) {

	restConfig, err := restconfig.RestConfig(env, oc)
	if err != nil {
		return nil, err
	}

	dh, err := dynamichelper.New(log, restConfig)
	if err != nil {
		return nil, err
	}

	configcli, err := configclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	fpAuth, err := env.FPAuthorizer(subscriptionDoc.Subscription.Properties.TenantID,
		env.Environment().ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	return &kubeActions{
		log:       log,
		oc:        oc,
		dh:        dh,
		configcli: configcli,

		virtualNetworks: network.NewVirtualNetworksClient(env.Environment(), subscriptionDoc.ID, fpAuth),
	}, nil
}

func (k *kubeActions) KubeGet(ctx context.Context, groupKind, namespace, name string) ([]byte, error) {
	un, err := k.dh.Get(ctx, groupKind, namespace, name)
	if err != nil {
		return nil, err
	}
	return un.MarshalJSON()
}

func (k *kubeActions) KubeList(ctx context.Context, groupKind, namespace string) ([]byte, error) {
	ul, err := k.dh.List(ctx, groupKind, namespace)
	if err != nil {
		return nil, err
	}
	return ul.MarshalJSON()
}

func (k *kubeActions) KubeCreateOrUpdate(ctx context.Context, obj *unstructured.Unstructured) error {
	return k.dh.CreateOrUpdate(ctx, obj)
}

func (k *kubeActions) KubeDelete(ctx context.Context, groupKind, namespace, name string) error {
	return k.dh.Delete(ctx, groupKind, namespace, name)
}
