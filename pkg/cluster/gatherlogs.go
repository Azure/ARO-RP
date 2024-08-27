package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/cluster/failurediagnostics"
	"github.com/Azure/ARO-RP/pkg/util/steps"
)

type diagnosticStep struct {
	f      func(context.Context) (interface{}, error)
	isJSON bool
}

func (m *manager) gatherFailureLogs(ctx context.Context) {
	d := failurediagnostics.NewFailureDiagnostics(m.log, m.env, m.doc, m.virtualMachines)

	for _, f := range []diagnosticStep{
		{f: m.logClusterVersion, isJSON: true},
		{f: m.logNodes, isJSON: true},
		{f: m.logClusterOperators, isJSON: true},
		{f: m.logIngressControllers, isJSON: true},
		{f: d.LogVMSerialConsole, isJSON: false},
		{f: m.logPodLogs, isJSON: false},
	} {
		o, err := f.f(ctx)
		if err != nil {
			m.log.Error(err)
			continue
		}

		if f.isJSON {
			b, err := json.MarshalIndent(o, "", "    ")
			if err != nil {
				m.log.Error(err)
				continue
			}

			m.log.Printf("%s: %s", steps.FriendlyName(f.f), string(b))
		} else {
			entries, ok := o.([]interface{})
			name := steps.FriendlyName(f.f)
			if ok {
				for _, i := range entries {
					m.log.Printf("%s: %v", name, i)
				}
			} else {
				m.log.Printf("%s: %v", steps.FriendlyName(f.f), o)
			}
		}
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

func (m *manager) logPodLogs(ctx context.Context) (interface{}, error) {
	if m.operatorcli == nil {
		return nil, nil
	}

	tailLines := int64(20)
	podLogOptions := corev1.PodLogOptions{
		TailLines: &tailLines,
	}
	items := make([]interface{}, 0)

	pods, err := m.kubernetescli.CoreV1().Pods("openshift-azure-operator").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, i := range pods.Items {
		items = append(items, fmt.Sprintf("pod status %s: %s", i.Name, i.Status))

		req := m.kubernetescli.CoreV1().Pods("openshift-azure-operator").GetLogs(i.Name, &podLogOptions)
		logForPod := m.log.WithField("pod", i.Name)
		logStream, err := req.Stream(ctx)
		if err != nil {
			items = append(items, fmt.Sprintf("pod logs retrieval error for %s: %s", i.Name, err))
			continue
		}
		defer logStream.Close()

		reader := bufio.NewReader(logStream)
		for {
			line, err := reader.ReadString('\n')
			logForPod.Debug(line)
			if err == io.EOF {
				break
			}
			if err != nil {
				m.log.Errorf("pod logs reading error for %s: %s", i.Name, err)
				break
			}
		}
	}
	return items, nil
}
