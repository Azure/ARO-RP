package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hiveclient "github.com/openshift/hive/pkg/client/clientset/versioned"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
)

type ClusterManager interface {
	CreateNamespace(ctx context.Context) (*corev1.Namespace, error)
	CreateOrUpdate(ctx context.Context, parameters *CreateOrUpdateParameters) error
	Delete(ctx context.Context, namespace string) error
	IsConnected(ctx context.Context, namespace string) (bool, string, error)
}

// CreateOrUpdateParameters represents all data in hive pertaining to a single ARO cluster.
// CreateOrUpdate must not receive any data which requires an API call to the customer cluster
// as the intention of this is to be able to reconcile hive resources from CosmosDB -> Hive
// and this process should work even if the customer cluster is not responding for any reason.
type CreateOrUpdateParameters struct {
	Namespace        string
	ClusterName      string
	Location         string
	InfraID          string
	ClusterID        string
	KubeConfig       string
	ServicePrincipal string
}

type clusterManager struct {
	hiveClientset *hiveclient.Clientset
	kubernetescli *kubernetes.Clientset

	dh dynamichelper.Interface
}

func NewClusterManagerFromConfig(log *logrus.Entry, restConfig *rest.Config) (ClusterManager, error) {
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

	return newClusterManager(log, hiveclientset, kubernetescli, dh), nil
}

func newClusterManager(log *logrus.Entry, hiveClientset *hiveclient.Clientset, kubernetescli *kubernetes.Clientset, dh dynamichelper.Interface) ClusterManager {
	return &clusterManager{
		hiveClientset: hiveClientset,
		kubernetescli: kubernetescli,

		dh: dh,
	}
}

func (hr *clusterManager) CreateNamespace(ctx context.Context) (*corev1.Namespace, error) {
	var namespaceName string
	var namespace *corev1.Namespace
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		namespaceName = "aro-" + uuid.Must(uuid.NewV4()).String()
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

func (hr *clusterManager) CreateOrUpdate(ctx context.Context, parameters *CreateOrUpdateParameters) error {
	resources := []kruntime.Object{
		kubeAdminSecret(parameters.Namespace, []byte(parameters.KubeConfig)),
		servicePrincipalSecret(parameters.Namespace, []byte(parameters.ServicePrincipal)),
		clusterDeployment(parameters.Namespace, parameters.ClusterName, parameters.ClusterID, parameters.InfraID, parameters.Location),
	}

	err := dynamichelper.Prepare(resources)
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
	// Just deleting the namespace for now
	return hr.kubernetescli.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
}

func (hr *clusterManager) IsConnected(ctx context.Context, namespace string) (bool, string, error) {
	cd, err := hr.hiveClientset.HiveV1().ClusterDeployments(namespace).Get(ctx, clusterDeploymentName, metav1.GetOptions{})
	if err != nil {
		return false, "", err
	}

	// Looking for the UnreachableCondition in the list of conditions
	// the order is not stable, but the condition is expected to be present
	for _, condition := range cd.Status.Conditions {
		if condition.Type == hivev1.UnreachableCondition {
			//checking for false, meaning not unreachable, so is reachable
			isReachable := condition.Status != corev1.ConditionTrue
			return isReachable, condition.Message, nil
		}
	}

	// we should never arrive here (famous last words)
	return false, "", fmt.Errorf("could not find UnreachableCondition")
}
