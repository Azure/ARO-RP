package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	kuval "k8s.io/apimachinery/pkg/util/validation"

	"github.com/Azure/ARO-RP/pkg/util/stringutils"
)

// Each cluster is being checked every 1 minute. emitDebugPodsCount
// lists all events (age < 120 seconds) in the default namespace
// in order to detect recent events of debug-pod creation.
func (mon *Monitor) emitDebugPodsCount(ctx context.Context) error {
	m := map[string]string{
		"reason":         "Started",
		"regarding.kind": "Pod",
		// oc debug node creates a pod with a "container-00"
		// container name.
		"regarding.fieldPath": "spec.containers{container-00}",
	}
	lo := metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(m).String(),
	}
	events, err := mon.cli.EventsV1().Events("default").List(ctx, lo)
	if err != nil {
		return err
	}

	nodes, err := mon.cli.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	podNames := getDebugPodNames(nodes.Items)

	// User can spawn a debug pod, make some activity and delete it
	// in the short timespan when the monitoring go routine is sleeping.
	// That's why we check the events of debug pod creation, not pods.
	count := countDebugPods(podNames, events.Items)
	mon.emitGauge("debugpods.count", int64(count), nil)

	return nil
}

func countDebugPods(debugPods []string, events []eventsv1.Event) int {
	count := 0
	for _, e := range events {
		if eventIsNew(e) && stringutils.Contains(debugPods, e.Regarding.Name) {
			count++
		}
	}
	return count
}

func eventIsNew(event eventsv1.Event) bool {
	return time.Since(event.Series.LastObservedTime.Time) < time.Second*120
}

func getDebugPodNames(nodes []corev1.Node) []string {
	names := make([]string, len(nodes))
	for i, n := range nodes {
		names[i] = fmt.Sprintf("%s-debug", MakeSimpleName(n.Name))
	}
	return names
}

// MakeSimpleName is a copy of the function that is used by oc CLI tool
// to generate names for the debug pods.
func MakeSimpleName(name string) string {
	invalidServiceChars := regexp.MustCompile("[^-a-z0-9]")

	name = strings.ToLower(name)
	name = invalidServiceChars.ReplaceAllString(name, "")
	name = strings.TrimFunc(name, func(r rune) bool { return r == '-' })
	if len(name) > kuval.DNS1035LabelMaxLength {
		name = name[:kuval.DNS1035LabelMaxLength]
	}
	return name
}
