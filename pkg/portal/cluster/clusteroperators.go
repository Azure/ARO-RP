package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1 "github.com/openshift/api/config/v1"
)

// Type for listing different operator conditions such as "progressing", "degraded" etc
type OperatorCondition struct {
	Type        configv1.ClusterStatusConditionType `json:"type"`
	LastUpdated string                              `json:"lastUpdated"`
	Status      configv1.ConditionStatus            `json:"status"`
	Reason      string                              `json:"reason"`
	Message     string                              `json:"message"`
}

// Type for holding information relating to a specific cluster operator. Certain conditions are listed
// as properties (Available, Progressing, Degraded) as they are default for all operators
type OperatorInformation struct {
	Name        string                   `json:"name"`
	Available   configv1.ConditionStatus `json:"available"`
	Progressing configv1.ConditionStatus `json:"progressing"`
	Degraded    configv1.ConditionStatus `json:"degraded"`
	Conditions  []OperatorCondition      `json:"conditions"`
}

type ClusterOperatorsInformation struct {
	Operators []OperatorInformation `json:"operators"`
}

func clusterOperatorsInformationFromOperatorList(operators *configv1.ClusterOperatorList) *ClusterOperatorsInformation {
	final := &ClusterOperatorsInformation{
		Operators: make([]OperatorInformation, 0, len(operators.Items)),
	}

	for _, co := range operators.Items {
		available := configv1.ConditionUnknown
		progressing := configv1.ConditionUnknown
		degraded := configv1.ConditionUnknown

		var conditions []OperatorCondition
		for _, cnd := range co.Status.Conditions {
			switch cnd.Type {
			case configv1.OperatorAvailable:
				available = cnd.Status
			case configv1.OperatorProgressing:
				progressing = cnd.Status
			case configv1.OperatorDegraded:
				degraded = cnd.Status
			}

			condition := OperatorCondition{
				Message:     cnd.Message,
				Reason:      cnd.Reason,
				Status:      cnd.Status,
				LastUpdated: cnd.LastTransitionTime.String(),
				Type:        cnd.Type,
			}

			conditions = append(conditions, condition)
		}

		final.Operators = append(final.Operators, OperatorInformation{
			Name:        co.Name,
			Available:   available,
			Progressing: progressing,
			Degraded:    degraded,
			Conditions:  conditions,
		})
	}

	return final
}

func (f *realFetcher) ClusterOperators(ctx context.Context) (*ClusterOperatorsInformation, error) {
	r, err := f.configCli.ConfigV1().ClusterOperators().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return clusterOperatorsInformationFromOperatorList(r), nil
}

func (c *client) ClusterOperators(ctx context.Context) (*ClusterOperatorsInformation, error) {
	return c.fetcher.ClusterOperators(ctx)
}
