package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hiveclient "github.com/openshift/hive/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
)

type ClusterManager interface {
	// We need to keep cluster SP updated with Hive, so we not only need to be able to create,
	// but also be able to update resources.
	// See relevant work item: https://msazure.visualstudio.com/AzureRedHatOpenShift/_workitems/edit/14895480
	// We might be able to do do this by using dynamic client.
	// Something similar to this: https://github.com/Azure/ARO-RP/pull/2145#discussion_r897915283
	// TODO: Replace Register with CreateOrUpdate and remove the comment above
	Register(ctx context.Context, workloadCluster *WorkloadCluster) (*hivev1.ClusterDeployment, error)
	Delete(ctx context.Context, namespace string) error
	IsConnected(ctx context.Context, namespace string) (bool, string, error)
}

// WorkloadCluster represents all data in hive pertaining to a single ARO cluster
type WorkloadCluster struct {
	SubscriptionID    string `json:"subscription,omitempty"`
	ClusterName       string `json:"name,omitempty"`
	ResourceGroupName string `json:"resourceGroup,omitempty"`
	Location          string `json:"location,omitempty"`
	InfraID           string `json:"infraId,omitempty"`
	ClusterID         string `json:"clusterID,omitempty"`
	KubeConfig        string `json:"kubeconfig,omitempty"`
	ServicePrincipal  string `json:"serviceprincipal,omitempty"`
}

type clusterManager struct {
	hiveClientset *hiveclient.Clientset
	kubernetescli *kubernetes.Clientset
}

func NewClusterManagerFromConfig(restConfig *rest.Config) (ClusterManager, error) {
	hiveclientset, err := hiveclient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	kubernetescli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return newClusterManager(hiveclientset, kubernetescli), nil
}

func newClusterManager(hiveClientset *hiveclient.Clientset, kubernetescli *kubernetes.Clientset) ClusterManager {
	return &clusterManager{
		hiveClientset: hiveClientset,
		kubernetescli: kubernetescli,
	}
}

func (hr *clusterManager) Register(ctx context.Context, workloadCluster *WorkloadCluster) (*hivev1.ClusterDeployment, error) {
	var namespace string

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		namespace = "aro-" + uuid.Must(uuid.NewV4()).String()
		csn := ClusterNamespace(namespace)
		_, err := hr.kubernetescli.CoreV1().Namespaces().Create(ctx, csn, metav1.CreateOptions{})
		return err
	})
	if err != nil {
		return nil, err
	}

	kubesecret := KubeAdminSecret(namespace, []byte(workloadCluster.KubeConfig))

	_, err = hr.kubernetescli.CoreV1().Secrets(namespace).Create(ctx, kubesecret, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	spsecret := ServicePrincipalSecret(namespace, []byte(workloadCluster.ServicePrincipal))
	_, err = hr.kubernetescli.CoreV1().Secrets(namespace).Create(ctx, spsecret, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	cds := ClusterDeployment(namespace, workloadCluster.ClusterName, workloadCluster.ClusterID, workloadCluster.InfraID, workloadCluster.Location)
	return hr.hiveClientset.HiveV1().ClusterDeployments(namespace).Create(ctx, cds, metav1.CreateOptions{})
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
