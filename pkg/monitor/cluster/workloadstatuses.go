package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"strconv"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubernetes/pkg/kubelet/events"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/ARO-RP/pkg/util/namespace"
)

var podConditionsExpected = map[corev1.PodConditionType]corev1.ConditionStatus{
	corev1.ContainersReady: corev1.ConditionTrue,
	corev1.PodInitialized:  corev1.ConditionTrue,
	corev1.PodScheduled:    corev1.ConditionTrue,
	corev1.PodReady:        corev1.ConditionTrue,
}

var restartCounterThreshold int32 = 10

func (mon *Monitor) emitWorkloadStatuses(ctx context.Context) error {
	// Phase 1: Get all pod metadata cluster-wide. This is a lightweight call.
	var totalPodCount, totalDaemonSetCount, totalDeploymentCount, totalReplicaSetCount, totalStatefulSetCount int64
	openshiftNamespacesWithPods := map[string]struct{}{}

	for _, gvk := range []schema.GroupVersionKind{
		{Group: "", Version: "v1", Kind: "PodList"},
		{Group: "apps", Version: "v1", Kind: "DaemonSetList"},
		{Group: "apps", Version: "v1", Kind: "DeploymentList"},
		{Group: "apps", Version: "v1", Kind: "ReplicaSetList"},
		{Group: "apps", Version: "v1", Kind: "StatefulSetList"},
	} {
		listOpts := &client.ListOptions{
			Limit: 500,
		}
		for {
			metaList := &metav1.PartialObjectMetadataList{}
			metaList.SetGroupVersionKind(gvk)

			err := mon.ocpclientset.List(ctx, metaList, listOpts)
			if err != nil {
				return err
			}

			switch gvk.Kind {
			case "PodList":
				totalPodCount += int64(len(metaList.Items))
			case "DaemonSetList":
				totalDaemonSetCount += int64(len(metaList.Items))
			case "DeploymentList":
				totalDeploymentCount += int64(len(metaList.Items))
			case "ReplicaSetList":
				totalReplicaSetCount += int64(len(metaList.Items))
			case "StatefulSetList":
				totalStatefulSetCount += int64(len(metaList.Items))
			}

			for _, p := range metaList.Items {
				if namespace.IsOpenShiftNamespace(p.Namespace) {
					openshiftNamespacesWithPods[p.Namespace] = struct{}{}
				}
			}

			if metaList.Continue == "" {
				break
			}
			listOpts.Continue = metaList.Continue
		}
	}

	mon.emitGauge("pod.count", totalPodCount, nil)
	mon.emitGauge("daemonset.count", totalDaemonSetCount, nil)
	mon.emitGauge("deployment.count", totalDeploymentCount, nil)
	mon.emitGauge("replicaset.count", totalReplicaSetCount, nil)
	mon.emitGauge("statefulset.count", totalStatefulSetCount, nil)

	// Phase 2: Get full pod objects only for the identified OpenShift namespaces.
	for ns := range openshiftNamespacesWithPods {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := mon.emitWorkloadStatusesForNamespace(ctx, ns)
		if err != nil {
			return err
		}
	}

	return nil
}

func (mon *Monitor) emitWorkloadStatusesForNamespace(ctx context.Context, ns string) error {
	// Pods
	var cont string
	for {
		ps, err := mon.cli.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{Limit: 500, Continue: cont})
		if err != nil {
			return err
		}

		mon._emitPodConditions(ps)
		mon._emitPodContainerStatuses(ps)
		mon._emitPodContainerRestartCounter(ps)

		cont = ps.Continue
		if cont == "" {
			break
		}
	}

	// DaemonSets
	cont = ""
	for {
		dss, err := mon.cli.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{Limit: 500, Continue: cont})
		if err != nil {
			return err
		}

		for _, ds := range dss.Items {
			if ds.Status.DesiredNumberScheduled == ds.Status.NumberAvailable {
				continue
			}

			mon.emitGauge("daemonset.statuses", 1, map[string]string{
				"desiredNumberScheduled": strconv.Itoa(int(ds.Status.DesiredNumberScheduled)),
				"name":                   ds.Name,
				"namespace":              ds.Namespace,
				"numberAvailable":        strconv.Itoa(int(ds.Status.NumberAvailable)),
			})
		}

		cont = dss.Continue
		if cont == "" {
			break
		}
	}

	// Deployments
	cont = ""
	for {
		ds, err := mon.cli.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{Limit: 500, Continue: cont})
		if err != nil {
			return err
		}

		for _, d := range ds.Items {
			if d.Status.Replicas == d.Status.AvailableReplicas {
				continue
			}

			mon.emitGauge("deployment.statuses", 1, map[string]string{
				"availableReplicas": strconv.Itoa(int(d.Status.AvailableReplicas)),
				"name":              d.Name,
				"namespace":         d.Namespace,
				"replicas":          strconv.Itoa(int(d.Status.Replicas)),
			})
		}

		cont = ds.Continue
		if cont == "" {
			break
		}
	}

	// ReplicaSets
	cont = ""
	for {
		rss, err := mon.cli.AppsV1().ReplicaSets(ns).List(ctx, metav1.ListOptions{Limit: 500, Continue: cont})
		if err != nil {
			return err
		}

		for _, rs := range rss.Items {
			if rs.Status.Replicas == rs.Status.AvailableReplicas {
				continue
			}

			mon.emitGauge("replicaset.statuses", 1, map[string]string{
				"availableReplicas": strconv.Itoa(int(rs.Status.AvailableReplicas)),
				"name":              rs.Name,
				"namespace":         rs.Namespace,
				"replicas":          strconv.Itoa(int(rs.Status.Replicas)),
			})
		}

		cont = rss.Continue
		if cont == "" {
			break
		}
	}

	// StatefulSets
	cont = ""
	for {
		sss, err := mon.cli.AppsV1().StatefulSets(ns).List(ctx, metav1.ListOptions{Limit: 500, Continue: cont})
		if err != nil {
			return err
		}

		for _, ss := range sss.Items {
			if ss.Status.Replicas == ss.Status.ReadyReplicas {
				continue
			}

			mon.emitGauge("statefulset.statuses", 1, map[string]string{
				"name":          ss.Name,
				"namespace":     ss.Namespace,
				"replicas":      strconv.Itoa(int(ss.Status.Replicas)),
				"readyReplicas": strconv.Itoa(int(ss.Status.ReadyReplicas)),
			})
		}

		cont = sss.Continue
		if cont == "" {
			break
		}
	}

	return nil
}

func (mon *Monitor) _emitPodConditions(ps *corev1.PodList) {
	for _, p := range ps.Items {
		if !namespace.IsOpenShiftNamespace(p.Namespace) {
			continue
		}

		if p.Status.Phase == corev1.PodSucceeded {
			continue
		}

		if p.Status.Reason == events.PreemptContainer {
			continue
		}

		for _, c := range p.Status.Conditions {
			if c.Status == podConditionsExpected[c.Type] {
				continue
			}

			mon.emitGauge("pod.conditions", 1, map[string]string{
				"name":      p.Name,
				"namespace": p.Namespace,
				"nodeName":  p.Spec.NodeName,
				"status":    string(c.Status),
				"type":      string(c.Type),
			})

			if mon.hourlyRun {
				mon.log.WithFields(logrus.Fields{
					"metric":    "pod.conditions",
					"name":      p.Name,
					"namespace": p.Namespace,
					"nodeName":  p.Spec.NodeName,
					"status":    c.Status,
					"type":      c.Type,
					"message":   c.Message,
				}).Print()
			}
		}
	}
}

func (mon *Monitor) _emitPodContainerStatuses(ps *corev1.PodList) {
	for _, p := range ps.Items {
		if !namespace.IsOpenShiftNamespace(p.Namespace) {
			continue
		}

		if p.Status.Phase == corev1.PodSucceeded {
			continue
		}

		for _, cs := range p.Status.ContainerStatuses {
			if cs.State.Waiting == nil {
				continue
			}

			containerStatus := map[string]string{
				"name":                 p.Name,
				"namespace":            p.Namespace,
				"nodeName":             p.Spec.NodeName,
				"containername":        cs.Name,
				"reason":               cs.State.Waiting.Reason,
				"lastTerminationState": "",
			}

			if cs.LastTerminationState.Terminated != nil {
				containerStatus["lastTerminationState"] = cs.LastTerminationState.Terminated.Reason
			}

			mon.emitGauge("pod.containerstatuses", 1, containerStatus)

			if mon.hourlyRun {
				logFields := logrus.Fields{
					"metric":  "pod.containerstatuses",
					"message": cs.State.Waiting.Message,
				}
				for label, labelVal := range containerStatus {
					logFields[label] = labelVal
				}
				mon.log.WithFields(logFields).Print()
			}
		}
	}
}

func (mon *Monitor) _emitPodContainerRestartCounter(ps *corev1.PodList) {
	for _, p := range ps.Items {
		if !namespace.IsOpenShiftNamespace(p.Namespace) {
			continue
		}

		//Sum up the total number of restarts in the pod to match the number of restarts shown in the 'oc get pods' display
		t := int32(0)
		for _, cs := range p.Status.ContainerStatuses {
			t += cs.RestartCount
		}

		if t < restartCounterThreshold {
			continue
		}

		mon.emitGauge("pod.restartcounter", int64(t), map[string]string{
			"name":      p.Name,
			"namespace": p.Namespace,
		})

		if mon.hourlyRun {
			mon.log.WithFields(logrus.Fields{
				"metric":    "pod.restartcounter",
				"name":      p.Name,
				"namespace": p.Namespace,
			}).Print()
		}
	}
}
