package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"reflect"

	"github.com/operator-framework/operator-sdk/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	"github.com/Azure/ARO-RP/pkg/operator"
	arov1alpha1 "github.com/Azure/ARO-RP/pkg/operator/apis/aro.openshift.io/v1alpha1"
	aroclient "github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/typed/aro.openshift.io/v1alpha1"
	"github.com/Azure/ARO-RP/pkg/util/version"
)

func setCondition(arocli aroclient.AroV1alpha1Interface, cond *status.Condition, role string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cluster, err := arocli.Clusters().Get(arov1alpha1.SingletonClusterName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		changed := cluster.Status.Conditions.SetCondition(*cond)

		if setStaticStatus(cluster, role) {
			changed = true
		}

		if !changed {
			return nil
		}

		_, err = arocli.Clusters().UpdateStatus(cluster)
		return err
	})
}

func setStaticStatus(cluster *arov1alpha1.Cluster, role string) (changed bool) {
	conditions := make(status.Conditions, 0, len(cluster.Status.Conditions))

	// cleanup any old conditions
	for _, cond := range cluster.Status.Conditions {
		switch cond.Type {
		case arov1alpha1.InternetReachableFromMaster, arov1alpha1.InternetReachableFromWorker:
			conditions = append(conditions, cond)
		default:
			changed = true
		}
	}

	cluster.Status.Conditions = conditions

	if role == operator.RoleMaster {
		relatedObjects := pullsecretRelatedObjects()
		cluster.Status.RelatedObjects = append(cluster.Status.RelatedObjects, genevaloggingRelatedObjects()...)
		cluster.Status.RelatedObjects = append(cluster.Status.RelatedObjects, alertwebhookRelatedObjects()...)

		if !reflect.DeepEqual(cluster.Status.RelatedObjects, relatedObjects) {
			cluster.Status.RelatedObjects = relatedObjects
			changed = true
		}

		if cluster.Status.OperatorVersion != version.GitCommit {
			cluster.Status.OperatorVersion = version.GitCommit
			changed = true
		}
	}

	return
}
