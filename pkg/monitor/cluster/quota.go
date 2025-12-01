package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"regexp"
	"time"

	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

const cpuQuotaMetric = "backend.openshiftcluster.quotareached.cpu"

func (mon *Monitor) detectQuotaFailure(ctx context.Context) error {
	m := map[string]string{
		"reason":         "FailedCreate",
		"regarding.kind": "Machine",
	}
	lo := metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(m).String(),
	}
	events, err := mon.cli.EventsV1().Events("openshift-machine-api").List(ctx, lo)
	if err != nil {
		return err
	}

	for _, event := range events.Items {
		if eventIsNew(event) && messageMatchesQuota(event.Note) {
			mon.emitGauge(cpuQuotaMetric, int64(1), nil)
			break
		}
	}

	return nil
}

func eventIsNew(e eventsv1.Event) bool {
	return time.Since(e.Series.LastObservedTime.Time) < time.Second*120
}

func messageMatchesQuota(message string) bool {
	re := regexp.MustCompile(`Operation could not be completed as it results in exceeding approved [a-zA-z0-9]+ Cores quota\. Additional details - Deployment Model: Resource Manager, Location: [a-zA-Z0-9\-]+, Current Limit: [0-9]+, Current Usage: [0-9]+, Additional Required: [0-9]+, \(Minimum\) New Limit Required: [0-9]+\.`)

	return re.MatchString(message)
}
