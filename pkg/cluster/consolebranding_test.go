package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1 "github.com/openshift/api/operator/v1"
	operatorfake "github.com/openshift/client-go/operator/clientset/versioned/fake"
)

func TestUpdateConsoleBranding(t *testing.T) {
	ctx := context.Background()

	consoleName := "cluster"

	m := &manager{
		log: logrus.NewEntry(logrus.StandardLogger()),
		operatorcli: operatorfake.NewSimpleClientset(&operatorv1.Console{
			ObjectMeta: metav1.ObjectMeta{
				Name: consoleName,
			},
			Status: operatorv1.ConsoleStatus{
				OperatorStatus: operatorv1.OperatorStatus{
					Conditions: []operatorv1.OperatorCondition{
						{
							Type:   "DeploymentAvailable",
							Status: operatorv1.ConditionTrue,
						},
					},
				},
			},
		}),
	}

	err := m.updateConsoleBranding(ctx)
	if err != nil {
		t.Error(err)
	}

	console, err := m.operatorcli.OperatorV1().Consoles().Get(ctx, consoleName, metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}

	if console.Spec.Customization.Brand != operatorv1.BrandAzure {
		t.Error(console.Spec.Customization.Brand)
	}
}
