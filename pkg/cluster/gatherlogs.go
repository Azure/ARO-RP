package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/ARO-RP/pkg/cluster/failurediagnostics"
	"github.com/Azure/ARO-RP/pkg/util/steps"
)

type diagnosticStep struct {
	f      func(context.Context) (interface{}, error)
	isJSON bool
}

func (m *manager) gatherFailureLogs(ctx context.Context, runType string) {
	d := failurediagnostics.NewFailureDiagnostics(m.log, m.env, m.doc, m.virtualMachines)

	s := []diagnosticStep{
		{f: m.logClusterVersion, isJSON: true},
		{f: m.logNodes, isJSON: true},
		{f: m.logClusterOperators, isJSON: true},
		{f: m.logIngressControllers, isJSON: true},
		{f: m.logPodLogs, isJSON: false},
	}

	// only log serial consoles on an install, not on updates/adminUpdates
	if runType == "install" {
		s = append(s, diagnosticStep{f: d.LogVMSerialConsole, isJSON: false})
	}

	for _, f := range s {
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

	lines := make([]string, 0)
	errs := make([]error, 0)

	for _, node := range nodes.Items {
		node.ManagedFields = nil

		nodeReady := corev1.ConditionUnknown
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady {
				nodeReady = condition.Status
				break
			}
		}
		lines = append(lines, fmt.Sprintf("%s - Ready: %s", node.Name, nodeReady))

		json, err := json.Marshal(node)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		m.log.Info(string(json))
	}

	return strings.Join(lines, "\n"), errors.Join(errs...)
}

func (m *manager) logClusterOperators(ctx context.Context) (interface{}, error) {
	if m.configcli == nil {
		return nil, nil
	}

	cos, err := m.configcli.ConfigV1().ClusterOperators().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	lines := make([]string, 0)
	errs := make([]error, 0)

	for _, co := range cos.Items {
		co.ManagedFields = nil

		coAvailable := configv1.ConditionUnknown
		coProgressing := configv1.ConditionUnknown
		coDegraded := configv1.ConditionUnknown
		for _, condition := range co.Status.Conditions {
			switch condition.Type {
			case configv1.OperatorAvailable:
				coAvailable = condition.Status
			case configv1.OperatorProgressing:
				coProgressing = condition.Status
			case configv1.OperatorDegraded:
				coDegraded = condition.Status
			}
		}
		lines = append(lines, fmt.Sprintf("%s - Available: %s, Progressing: %s, Degraded: %s", co.Name, coAvailable, coProgressing, coDegraded))

		json, err := json.Marshal(co)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		m.log.Infof(string(json))
	}

	return strings.Join(lines, "\n"), errors.Join(errs...)
}

func (m *manager) logIngressControllers(ctx context.Context) (interface{}, error) {
	if m.operatorcli == nil {
		return nil, nil
	}

	ics, err := m.operatorcli.OperatorV1().IngressControllers("openshift-ingress-operator").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	lines := make([]string, 0)
	errs := make([]error, 0)

	for _, ic := range ics.Items {
		ic.ManagedFields = nil

		icAvailable := operatorv1.ConditionUnknown
		icProgressing := operatorv1.ConditionUnknown
		icDegraded := operatorv1.ConditionUnknown
		for _, condition := range ic.Status.Conditions {
			switch condition.Type {
			case operatorv1.OperatorStatusTypeAvailable:
				icAvailable = condition.Status
			case operatorv1.OperatorStatusTypeProgressing:
				icProgressing = condition.Status
			case operatorv1.OperatorStatusTypeDegraded:
				icDegraded = condition.Status
			}
		}
		lines = append(lines, fmt.Sprintf("%s - Available: %s, Progressing: %s, Degraded: %s", ic.Name, icAvailable, icProgressing, icDegraded))

		json, err := json.Marshal(ic)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		m.log.Infof(string(json))
	}

	return strings.Join(lines, "\n"), errors.Join(errs...)
}

func (m *manager) logPodLogs(ctx context.Context) (interface{}, error) {
	if m.kubernetescli == nil {
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
		items = append(items, fmt.Sprintf("pod status %s: %v", i.Name, i.Status))

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
			logForPod.Info(strings.TrimSpace(line))
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
