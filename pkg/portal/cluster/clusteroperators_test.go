package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"sort"
	"testing"

	"github.com/go-test/deep"
	configv1 "github.com/openshift/api/config/v1"
	configfake "github.com/openshift/client-go/config/clientset/versioned/fake"
	kruntime "k8s.io/apimachinery/pkg/runtime"

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

	// lol golang
	converted := make([]kruntime.Object, len(operators.Items))
	for i := range operators.Items {
		converted[i] = &operators.Items[i]
	}

	configcli := configfake.NewSimpleClientset(converted...)

	_, log := testlog.New()

	rf := &realFetcher{
		configcli: configcli,
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
				Name:      "authentication",
				Available: "True",
			},
			{
				Name:      "cluster-autoscaler",
				Available: "True",
			},
			{
				Name:      "cloud-credential",
				Available: "True",
			},
			{
				Name:      "config-operator",
				Available: "True",
			},
			{
				Name:      "console",
				Available: "True",
			},
			{
				Name:      "aro",
				Available: "True",
			},
		},
	}

	sort.SliceStable(info.Operators, func(i, j int) bool { return info.Operators[i].Name < info.Operators[j].Name })
	sort.SliceStable(expected.Operators, func(i, j int) bool { return expected.Operators[i].Name < expected.Operators[j].Name })

	for _, r := range deep.Equal(expected, info) {
		t.Error(r)
	}
}
