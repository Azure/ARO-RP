package hive

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hiveclient "github.com/openshift/hive/pkg/client/clientset/versioned"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/api"
	"github.com/Azure/ARO-RP/pkg/env"
	"github.com/Azure/ARO-RP/pkg/util/dynamichelper"
	utillog "github.com/Azure/ARO-RP/pkg/util/log"
	"github.com/Azure/ARO-RP/pkg/util/uuid"
)

type ClusterManager interface {
	CreateNamespace(ctx context.Context) (*corev1.Namespace, error)

	// CreateOrUpdate reconciles the ClusterDocument and related secrets for an
	// existing cluster. This may adopt the cluster (Create) or amend the
	// existing resources (Update).
	CreateOrUpdate(ctx context.Context, sub *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument) error
	// Delete removes the cluster from Hive.
	Delete(ctx context.Context, doc *api.OpenShiftClusterDocument) error
	// Install creates a ClusterDocument and related secrets for a new cluster
	// so that it can be provisioned by Hive.
	Install(ctx context.Context, sub *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument, version *api.OpenShiftVersion) error
	IsClusterDeploymentReady(ctx context.Context, doc *api.OpenShiftClusterDocument) (bool, error)
	IsClusterInstallationComplete(ctx context.Context, doc *api.OpenShiftClusterDocument) (bool, error)
	GetClusterDeployment(ctx context.Context, doc *api.OpenShiftClusterDocument) (*hivev1.ClusterDeployment, error)
	ResetCorrelationData(ctx context.Context, doc *api.OpenShiftClusterDocument) error
}

type clusterManager struct {
	log *logrus.Entry
	env env.Core

	hiveClientset hiveclient.Interface
	kubernetescli kubernetes.Interface

	dh dynamichelper.Interface
}

// NewFromEnv can return a nil ClusterManager when hive features are disabled. This exists to support regions where we don't have hive,
// and we do not want to restrict the frontend from starting up successfully.
// It has the caveat of requiring a nil check on any operations performed with the returned ClusterManager
// until this conditional return is removed (we have hive everywhere).
func NewFromEnv(ctx context.Context, log *logrus.Entry, env env.Interface) (ClusterManager, error) {
	adoptByHive, err := env.LiveConfig().AdoptByHive(ctx)
	if err != nil {
		return nil, err
	}
	installViaHive, err := env.LiveConfig().InstallViaHive(ctx)
	if err != nil {
		return nil, err
	}
	if !adoptByHive && !installViaHive {
		log.Infof("hive is disabled, skipping creation of ClusterManager")
		return nil, nil
	}
	hiveShard := 1
	hiveRestConfig, err := env.LiveConfig().HiveRestConfig(ctx, hiveShard)
	if err != nil {
		return nil, fmt.Errorf("failed getting RESTConfig for Hive shard %d: %w", hiveShard, err)
	}
	return NewFromConfig(log, env, hiveRestConfig)
}

// NewFromConfig creates a ClusterManager.
// It MUST NOT take cluster or subscription document as values
// in these structs can be change during the lifetime of the cluster manager.
func NewFromConfig(log *logrus.Entry, _env env.Core, restConfig *rest.Config) (ClusterManager, error) {
	hiveClientset, err := hiveclient.NewForConfig(restConfig)
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

	return &clusterManager{
		log: log,
		env: _env,

		hiveClientset: hiveClientset,
		kubernetescli: kubernetescli,

		dh: dh,
	}, nil
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

func (hr *clusterManager) CreateOrUpdate(ctx context.Context, sub *api.SubscriptionDocument, doc *api.OpenShiftClusterDocument) error {
	resources, err := hr.resources(sub, doc)
	if err != nil {
		return err
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

func (hr *clusterManager) Delete(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	err := hr.kubernetescli.CoreV1().Namespaces().Delete(ctx, doc.OpenShiftCluster.Properties.HiveProfile.Namespace, metav1.DeleteOptions{})
	if err != nil && kerrors.IsNotFound(err) {
		return nil
	}

	return err
}

func (hr *clusterManager) IsClusterDeploymentReady(ctx context.Context, doc *api.OpenShiftClusterDocument) (bool, error) {
	cd, err := hr.GetClusterDeployment(ctx, doc)
	if err != nil {
		return false, err
	}

	if len(cd.Status.Conditions) == 0 {
		return false, nil
	}

	checkConditions := map[hivev1.ClusterDeploymentConditionType]corev1.ConditionStatus{
		hivev1.ProvisionedCondition:                     corev1.ConditionTrue,
		hivev1.SyncSetFailedCondition:                   corev1.ConditionFalse,
		hivev1.ControlPlaneCertificateNotFoundCondition: corev1.ConditionFalse,
		hivev1.UnreachableCondition:                     corev1.ConditionFalse,
	}

	for _, cond := range cd.Status.Conditions {
		conditionStatus, found := checkConditions[cond.Type]
		if found && conditionStatus != cond.Status {
			hr.log.Infof("clusterdeployment not ready: %s == %s", cond.Type, cond.Status)
			return false, nil
		}
	}

	return true, nil
}

func (hr *clusterManager) IsClusterInstallationComplete(ctx context.Context, doc *api.OpenShiftClusterDocument) (bool, error) {
	cd, err := hr.hiveClientset.HiveV1().ClusterDeployments(doc.OpenShiftCluster.Properties.HiveProfile.Namespace).Get(ctx, ClusterDeploymentName, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	if cd.Spec.Installed {
		return true, nil
	}

	checkFailureConditions := map[hivev1.ClusterDeploymentConditionType]corev1.ConditionStatus{
		hivev1.ProvisionFailedCondition: corev1.ConditionTrue,
	}

	for _, cond := range cd.Status.Conditions {
		conditionStatus, found := checkFailureConditions[cond.Type]
		if found && conditionStatus == cond.Status {
			return false, fmt.Errorf("clusterdeployment has failed: %s == %s", cond.Type, cond.Status)
		}
	}

	return false, nil
}

func (hr *clusterManager) GetClusterDeployment(ctx context.Context, doc *api.OpenShiftClusterDocument) (*hivev1.ClusterDeployment, error) {
	return hr.hiveClientset.HiveV1().ClusterDeployments(doc.OpenShiftCluster.Properties.HiveProfile.Namespace).Get(ctx, ClusterDeploymentName, metav1.GetOptions{})
}

func (hr *clusterManager) ResetCorrelationData(ctx context.Context, doc *api.OpenShiftClusterDocument) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cd, err := hr.hiveClientset.HiveV1().ClusterDeployments(doc.OpenShiftCluster.Properties.HiveProfile.Namespace).Get(ctx, ClusterDeploymentName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		err = utillog.ResetHiveCorrelationData(cd)
		if err != nil {
			return err
		}

		_, err = hr.hiveClientset.HiveV1().ClusterDeployments(doc.OpenShiftCluster.Properties.HiveProfile.Namespace).Update(ctx, cd, metav1.UpdateOptions{})
		return err
	})
}
