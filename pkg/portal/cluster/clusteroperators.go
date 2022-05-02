package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type OperatorInformation struct {
	Name      string                   `json:"name"`
	Available configv1.ConditionStatus `json:"available"`
}

type ClusterOperatorsInformation struct {
	Operators []OperatorInformation `json:"operators"`
}

func clusterOperatorsInformationFromOperatorList(operators *configv1.ClusterOperatorList) *ClusterOperatorsInformation {
	final := &ClusterOperatorsInformation{
		Operators: make([]OperatorInformation, 0, len(operators.Items)),
	}

	for _, co := range operators.Items {
		var Available = configv1.ConditionUnknown

		for _, cnd := range co.Status.Conditions {
			if cnd.Type == "Available" {
				Available = cnd.Status
			}
		}

		final.Operators = append(final.Operators, OperatorInformation{
			Name:      co.Name,
			Available: Available,
		})
	}

	return final
}

func (f *realFetcher) ClusterOperators(ctx context.Context) (*ClusterOperatorsInformation, error) {
	r, err := f.configcli.ConfigV1().ClusterOperators().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return clusterOperatorsInformationFromOperatorList(r), nil
}

func (c *client) ClusterOperators(ctx context.Context) (*ClusterOperatorsInformation, error) {
	return c.fetcher.ClusterOperators(ctx)
}
