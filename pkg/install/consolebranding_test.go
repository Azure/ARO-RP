package install

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"

	operatorv1 "github.com/openshift/api/operator/v1"
	v1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/client-go/operator/clientset/versioned/fake"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateConsoleBranding(t *testing.T) {
	ctx := context.Background()

	consoleName := "cluster"

	i := &Installer{
		log: logrus.NewEntry(logrus.StandardLogger()),
		operatorcli: fake.NewSimpleClientset(&v1.Console{
			ObjectMeta: metav1.ObjectMeta{
				Name: consoleName,
			},
			Status: v1.ConsoleStatus{
				OperatorStatus: v1.OperatorStatus{
					Conditions: []v1.OperatorCondition{
						{
							Type:   "DeploymentAvailable",
							Status: v1.ConditionTrue,
						},
					},
				},
			},
		}),
	}

	err := i.updateConsoleBranding(ctx)
	if err != nil {
		t.Error(err)
	}

	console, err := i.operatorcli.OperatorV1().Consoles().Get(consoleName, metav1.GetOptions{})
	if err != nil {
		t.Error(err)
	}

	if console.Spec.Customization.Brand != operatorv1.BrandAzure {
		t.Error(console.Spec.Customization.Brand)
	}
}
