package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	testmonitor "github.com/Azure/ARO-RP/test/util/monitor"
)

func TestMHCQuota2(t *testing.T) {
	testCases := []struct {
		name         string
		eventTime    time.Time
		eventMessage string
		wantGauge    bool
	}{
		{
			name:         "mhc failed because of quota",
			wantGauge:    true,
			eventMessage: "something something Operation could not be completed as it results in exceeding approved standardMSFamily Cores quota. Additional details - Deployment Model: Resource Manager, Location: eastus-2, Current Limit: 0, Current Usage: 0, Additional Required: 128, (Minimum) New Limit Required: 128. and some more things",
			eventTime:    time.Now(),
		},
		{
			name:         "mhc failed because of something else",
			wantGauge:    false,
			eventMessage: "not today!",
			eventTime:    time.Now(),
		},
		{
			name:         "mhc failed because of quota but old",
			wantGauge:    false,
			eventMessage: "something something Operation could not be completed as it results in exceeding approved standardMSFamily Cores quota. Additional details - Deployment Model: Resource Manager, Location: eastus-2, Current Limit: 0, Current Usage: 0, Additional Required: 128, (Minimum) New Limit Required: 128. and some more things",
			eventTime:    time.Date(1789, time.July, 14, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			cli := fake.NewSimpleClientset()

			cli.PrependReactor("list", "events",
				func(_ ktesting.Action) (handled bool, ret kruntime.Object, err error) {
					return true, &eventsv1.EventList{Items: []eventsv1.Event{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "something failed because of quota",
								Namespace: "openshift-machine-api",
							},
							Reason: "Started",
							Regarding: corev1.ObjectReference{
								Kind: "Machine",
								Name: "supermachine-1",
							},
							Note: tt.eventMessage,

							Series: &eventsv1.EventSeries{
								LastObservedTime: metav1.NewMicroTime(tt.eventTime),
							},
						},
					}}, nil
				},
			)

			m := testmonitor.NewFakeEmitter(t)
			mon := &Monitor{
				cli: cli,
				m:   m,
			}

			err := mon.detectQuotaFailure(ctx)
			if err != nil {
				t.Fatalf("got unexpected error: %v", err)
			}

			if tt.wantGauge {
				m.VerifyEmittedMetrics(testmonitor.Metric(cpuQuotaMetric, int64(1), map[string]string{}))
			} else {
				m.VerifyEmittedMetrics()
			}
		})
	}
}
