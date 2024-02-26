package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/util/steps"
)

func (m *manager) gatherFailureLogs(ctx context.Context) {
	for _, f := range []func(context.Context) (interface{}, error){
		m.logClusterVersion,
		m.logNodes,
		m.logClusterOperators,
		m.logIngressControllers,
	} {
		o, err := f(ctx)
		if err != nil {
			m.log.Error(err)
			continue
		}

		b, err := json.MarshalIndent(o, "", "    ")
		if err != nil {
			m.log.Error(err)
			continue
		}

		m.log.Printf("%s: %s", steps.ShortFriendlyName(f), string(b))
	}
}

func (m *manager) logClusterVersion(ctx context.Context) (interface{}, error) {
	if m.configcli == nil {
		return nil, nil
	}

	cv, err := m.configcli.ConfigV1().ClusterVersions().Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	cv.ManagedFields = nil

	return cv, nil
}

func (m *manager) logNodes(ctx context.Context) (interface{}, error) {
	if m.kubernetescli == nil {
		return nil, nil
	}

	nodes, err := m.kubernetescli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for i := range nodes.Items {
		nodes.Items[i].ManagedFields = nil
	}

	return nodes.Items, nil
}

func (m *manager) logClusterOperators(ctx context.Context) (interface{}, error) {
	if m.configcli == nil {
		return nil, nil
	}

	cos, err := m.configcli.ConfigV1().ClusterOperators().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for i := range cos.Items {
		cos.Items[i].ManagedFields = nil
	}

	return cos.Items, nil
}

func (m *manager) logIngressControllers(ctx context.Context) (interface{}, error) {
	if m.operatorcli == nil {
		return nil, nil
	}

	ics, err := m.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for i := range ics.Items {
		ics.Items[i].ManagedFields = nil
	}

	return ics.Items, nil
}
