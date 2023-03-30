package internetchecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	consolev1 "github.com/openshift/api/console/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	consoleclient "github.com/openshift/client-go/console/clientset/versioned"
	"github.com/openshift/library-go/pkg/config/clusteroperator/v1helpers"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/ARO-RP/pkg/operator"
)

const (
	consoleBannerName   = "aro-sre-unsupported-condition"
	clusterOperatorName = "aro"
	reasonAsExpected    = "AsExpected"

	WORKER_NODE_MINIMUM_COUNT = 3

	OperatorUpgradableFalse = configv1.ConditionFalse
	OperatorUpgradableTrue  = configv1.ConditionTrue
)

type UnsupportedConditionChecker struct {
	log *logrus.Entry

	kubernetescli kubernetes.Interface
	consolecli    consoleclient.Interface
	configcli     configclient.Interface

	role string
}

func NewUnsupportedConditionChecker(log *logrus.Entry, kubernetescli kubernetes.Interface, consolecli consoleclient.Interface, configcli configclient.Interface, role string) *UnsupportedConditionChecker {
	return &UnsupportedConditionChecker{
		log:           log,
		kubernetescli: kubernetescli,
		consolecli:    consolecli,
		configcli:     configcli,
		role:          role,
	}
}

func (ucc *UnsupportedConditionChecker) Name() string {
	return "UnsupportedConditionChecker"
}

func (ucc *UnsupportedConditionChecker) Check(ctx context.Context) error {
	if ucc.role != operator.RoleMaster {
		return nil
	}

	return ucc.checkWorkerNodeCount(ctx)
}

func (ucc *UnsupportedConditionChecker) checkWorkerNodeCount(ctx context.Context) error {
	nodes, err := ucc.kubernetescli.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/worker",
	})
	if err != nil {
		return err
	}

	if len(nodes.Items) < WORKER_NODE_MINIMUM_COUNT {
		err = ucc.setClusterOperatorStatus(ctx, OperatorUpgradableFalse)
		if err != nil {
			return err
		}

		return ucc.addConsoleBanner(ctx, fmt.Sprintf("Unsupported Cluster State: There needs to be at least %d worker nodes running", WORKER_NODE_MINIMUM_COUNT))
	}

	err = ucc.setClusterOperatorStatus(ctx, OperatorUpgradableTrue)
	if err != nil {
		return err
	}

	return ucc.consolecli.ConsoleV1().ConsoleNotifications().Delete(ctx, consoleBannerName, metav1.DeleteOptions{})
}

func (ucc *UnsupportedConditionChecker) setClusterOperatorStatus(ctx context.Context, upgradeableStatus configv1.ConditionStatus) error {
	co, err := ucc.configcli.ConfigV1().ClusterOperators().Get(ctx, clusterOperatorName, metav1.GetOptions{})

	if err != nil {
		ucc.log.Errorf("Failed to get the %q operator: %q", clusterOperatorName, err.Error())
		return err
	}

	condition := configv1.ClusterOperatorStatusCondition{
		Type:               configv1.OperatorUpgradeable,
		Status:             upgradeableStatus,
		LastTransitionTime: metav1.Now(),
		Reason:             reasonAsExpected,
	}

	v1helpers.SetStatusCondition(&co.Status.Conditions, condition)

	_, err = ucc.configcli.ConfigV1().ClusterOperators().UpdateStatus(ctx, co, metav1.UpdateOptions{})
	if err != nil {
		ucc.log.Errorf("Failed to update the %q operator: %q", clusterOperatorName, err.Error())
		return err
	}

	ucc.log.Infof("Set the %q operator upgradeable status to %q", clusterOperatorName, upgradeableStatus)

	return nil
}

func (ucc *UnsupportedConditionChecker) addConsoleBanner(ctx context.Context, consoleText string) error {
	notification, err := ucc.consolecli.ConsoleV1().ConsoleNotifications().Get(ctx, consoleBannerName, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		newBanner := &consolev1.ConsoleNotification{
			ObjectMeta: metav1.ObjectMeta{
				Name: consoleBannerName,
			},
			Spec: consolev1.ConsoleNotificationSpec{
				Text:            consoleText,
				Location:        consolev1.BannerTop,
				Color:           "#000",
				BackgroundColor: "#ff0",
			},
		}
		_, err = ucc.consolecli.ConsoleV1().ConsoleNotifications().Create(ctx, newBanner, metav1.CreateOptions{})
		if err != nil {
			ucc.log.Errorf("Failed to create the %q banner`: %q", consoleBannerName, err.Error())
			return err
		}
		ucc.log.Infof("Created the %q banner, with text %q", consoleBannerName, consoleText)
		return nil
	}
	if err != nil {
		return err
	}

	notification.Spec.Text = consoleText
	_, err = ucc.consolecli.ConsoleV1().ConsoleNotifications().Update(ctx, notification, metav1.UpdateOptions{})
	if err != nil {
		ucc.log.Errorf("Failed to update the %q banner`: %q", consoleBannerName, err.Error())
		return err
	}
	ucc.log.Infof("Updated the %q banner, with combined text %q", consoleBannerName, consoleText)
	return nil
}
