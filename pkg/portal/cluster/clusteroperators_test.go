package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"sort"
	"testing"

	"github.com/go-test/deep"

	kruntime "k8s.io/apimachinery/pkg/runtime"

	configv1 "github.com/openshift/api/config/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"

	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestClusterOperators(t *testing.T) {
	ctx := context.Background()

	txt, _ := clusteroperatorJsonBytes()

	var operators configv1.ClusterOperatorList
	err := json.Unmarshal(txt, &operators)
	if err != nil {
		t.Error(err)
	}

	converted := make([]kruntime.Object, len(operators.Items))
	for i := range operators.Items {
		converted[i] = &operators.Items[i]
	}

	configCli := configfake.NewSimpleClientset(converted...)

	_, log := testlog.New()

	rf := &realFetcher{
		configCli: configCli,
		log:       log,
	}

	c := &client{fetcher: rf, log: log}

	info, err := c.ClusterOperators(ctx)
	if err != nil {
		t.Error(err)
		return
	}

	expected := &ClusterOperatorsInformation{
		Operators: []OperatorInformation{
			{
				Name:        "authentication",
				Available:   "True",
				Degraded:    "False",
				Progressing: "False",
				Conditions: []OperatorCondition{
					{
						Reason: "AsExpected",
						Status: "False",
						Type:   "Degraded",
					},
					{
						Reason: "AsExpected",
						Status: "False",
						Type:   "Progressing",
					},
					{
						Message: "OAuthServerDeploymentAvailable: availableReplicas==2",
						Reason:  "AsExpected",
						Status:  "True",
						Type:    "Available",
					},
					{
						Reason: "AsExpected",
						Status: "True",
						Type:   "Upgradeable",
					},
				},
			},
			{
				Name:        "cluster-autoscaler",
				Available:   "True",
				Degraded:    "False",
				Progressing: "False",
				Conditions: []OperatorCondition{
					{
						Message: "at version 4.6.26",
						Reason:  "AsExpected",
						Status:  "True",
						Type:    "Available",
					},
					{
						Status: "False",
						Type:   "Progressing",
					},
					{
						Status: "False",
						Type:   "Degraded",
					},
					{
						Status: "True",
						Type:   "Upgradeable",
					},
				},
			},
			{
				Name:        "cloud-credential",
				Available:   "True",
				Degraded:    "False",
				Progressing: "False",
				Conditions: []OperatorCondition{
					{
						Status: "True",
						Type:   "Available",
					},
					{
						Status: "False",
						Type:   "Degraded",
					},
					{
						Status: "False",
						Type:   "Progressing",
					},
					{
						Status: "True",
						Type:   "Upgradeable",
					},
				},
			},
			{
				Name:        "config-operator",
				Available:   "True",
				Degraded:    "False",
				Progressing: "False",
				Conditions: []OperatorCondition{
					{
						Reason: "AsExpected",
						Status: "False",
						Type:   "Degraded",
					},
					{
						Reason: "AsExpected",
						Status: "False",
						Type:   "Progressing",
					},
					{
						Reason: "AsExpected",
						Status: "True",
						Type:   "Available",
					},
					{
						Reason: "AsExpected",
						Status: "True",
						Type:   "Upgradeable",
					},
				},
			},
			{
				Name:        "console",
				Available:   "True",
				Degraded:    "False",
				Progressing: "False",
				Conditions: []OperatorCondition{
					{
						Reason: "AsExpected",
						Status: "False",
						Type:   "Degraded",
					},
					{
						Reason: "AsExpected",
						Status: "False",
						Type:   "Progressing",
					},
					{
						Reason: "AsExpected",
						Status: "True",
						Type:   "Available",
					},
					{
						Reason: "AsExpected",
						Status: "True",
						Type:   "Upgradeable",
					},
				},
			},
			{
				Name:        "aro",
				Available:   "True",
				Degraded:    "False",
				Progressing: "False",
				Conditions: []OperatorCondition{
					{
						Reason: "AsExpected",
						Status: "True",
						Type:   "Available",
					},
					{
						Reason: "AsExpected",
						Status: "False",
						Type:   "Progressing",
					},
					{
						Reason: "AsExpected",
						Status: "False",
						Type:   "Degraded",
					},
					{
						Reason: "AsExpected",
						Status: "True",
						Type:   "Upgradeable",
					},
				},
			},
		},
	}

	for i, r := range info.Operators {
		for j := range r.Conditions {
			info.Operators[i].Conditions[j].LastUpdated = ""
		}
	}

	sort.SliceStable(info.Operators, func(i, j int) bool { return info.Operators[i].Name < info.Operators[j].Name })
	sort.SliceStable(expected.Operators, func(i, j int) bool { return expected.Operators[i].Name < expected.Operators[j].Name })

	for _, r := range deep.Equal(expected, info) {
		t.Error(r)
	}
}
