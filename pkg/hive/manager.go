package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	hiveclient "github.com/openshift/hive/pkg/client/clientset/versioned"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

type ClusterManager interface {
	CreateNamespace(ctx context.Context) (*corev1.Namespace, error)
	CreateOrUpdate(ctx context.Context) error
	Delete(ctx context.Context, namespace string) error
}

type clusterManager struct {
	subscriptionDoc *api.SubscriptionDocument
	doc             *api.OpenShiftClusterDocument

	hiveClientset *hiveclient.Clientset
	kubernetescli *kubernetes.Clientset

	dh dynamichelper.Interface
}

func NewFromConfig(log *logrus.Entry, subscriptionDoc *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument, restConfig *rest.Config) (ClusterManager, error) {
	hiveclientset, err := hiveclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	kubernetescli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	dh, err := dynamichelper.New(log, restConfig)
	if err != nil {
		return nil, err
	}

	return new(subscriptionDoc, doc, hiveclientset, kubernetescli, dh), nil
}

func new(subscriptionDoc *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument, hiveClientset *hiveclient.Clientset, kubernetescli *kubernetes.Clientset, dh dynamichelper.Interface) ClusterManager {
	return &clusterManager{
		subscriptionDoc: subscriptionDoc,
		doc:             doc,

		hiveClientset: hiveClientset,
		kubernetescli: kubernetescli,

		dh: dh,
	}
}

func (hr *clusterManager) CreateNamespace(ctx context.Context) (*corev1.Namespace, error) {
	var namespaceName string
	var namespace *corev1.Namespace
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		namespaceName = "aro-" + uuid.DefaultGenerator.Generate()
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
			},
		}

		var err error // Don't shadow namespace variable
		namespace, err = hr.kubernetescli.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
		return err
	})
	if err != nil {
		return nil, err
	}

	return namespace, nil
}

func (hr *clusterManager) CreateOrUpdate(ctx context.Context) error {
	namespace := hr.doc.OpenShiftCluster.Properties.HiveProfile.Namespace

	clusterSP, err := clusterSPToBytes(hr.subscriptionDoc, hr.doc.OpenShiftCluster)
	if err != nil {
		return err
	}

	resources := []kruntime.Object{
		aroServiceKubeconfigSecret(namespace, hr.doc.OpenShiftCluster.Properties.AROServiceKubeconfig),
		clusterServicePrincipalSecret(namespace, clusterSP),
		clusterDeployment(
			namespace,
			hr.doc.OpenShiftCluster.Name,
			hr.doc.ID,
			hr.doc.OpenShiftCluster.Properties.InfraID,
			hr.doc.OpenShiftCluster.Location,
			hr.doc.OpenShiftCluster.Properties.NetworkProfile.APIServerPrivateEndpointIP,
		),
	}

	err = dynamichelper.Prepare(resources)
	if err != nil {
		return err
	}

	err = hr.dh.Ensure(ctx, resources...)
	if err != nil {
		return err
	}

	return nil
}

func (hr *clusterManager) Delete(ctx context.Context, namespace string) error {
	err := hr.kubernetescli.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
	if err != nil && kerrors.IsNotFound(err) {
		return nil
	}

	return err
}
