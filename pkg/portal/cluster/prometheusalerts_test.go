package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/golang/mock/gomock"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"

	mockAlertManager "github.com/Azure/ARO-RP/pkg/util/mocks/alertmanager"
)

func TestFiringAlerts(t *testing.T) {
	ctx := context.Background()

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	entry := logrus.NewEntry(logger)

	for _, tt := range []struct {
		name          string
		returedData   []model.Alert
		expected      []Alert
		errorExpected error
	}{
		{
			name: "return firing alert",
			returedData: []model.Alert{
				{
					Labels: model.LabelSet{
						"alertname": "Firing Alert 1",
						"namespace": "openshift-apiserver",
						"status":    "firing",
						"severity":  "Info",
						"summary":   "summary of the test alert",
					},
					StartsAt: time.Now().Add(-1),
				},
				{
					Labels: model.LabelSet{
						"alertname": "Resolved Alert 1",
						"namespace": "openshift-apiserver",
					},
					StartsAt: time.Now().Add(-1),
					EndsAt:   time.Now().Add(-1),
				},
			},
			expected: []Alert{
				{
					AlertName: "Firing Alert 1",
					Status:    "firing",
					Namespace: "openshift-apiserver",
					Severity:  "Info",
					Summary:   "",
				},
			},
		},
		{
			name: "return no firing alerts for other namespaces",
			returedData: []model.Alert{
				{
					Labels: model.LabelSet{
						"alertname": "Firing Alert 1",
						"namespace": "other-namespace",
					},
					StartsAt: time.Now().Add(-1),
				},
				{
					Labels: model.LabelSet{
						"alertname": "Firing Alert2",
						"namespace": "some-other-namespace",
					},
					StartsAt: time.Now().Add(-1),
				},
			},
			expected: []Alert{},
		},
		{
			name:          "handles error gracefully",
			returedData:   []model.Alert{},
			expected:      []Alert{},
			errorExpected: fmt.Errorf("some error"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			mockAlertManagerClient := mockAlertManager.NewMockAlertManager(controller)

			mockAlertManagerClient.EXPECT().FetchPrometheusAlerts(ctx, alertmanagerService).AnyTimes().Return(tt.returedData, tt.errorExpected)

			rf := &realFetcher{
				alertManagerClient: mockAlertManagerClient,
				log:                entry,
			}

			c := &client{fetcher: rf, log: entry}

			alerts, err := c.GetOpenShiftFiringAlerts(ctx)
			if err != nil && !strings.EqualFold(tt.errorExpected.Error(), err.Error()) {
				t.Error(err)
				return
			}

			// Don't run deep equal if both of the slices have a length of zero
			if len(tt.returedData) > 0 || len(tt.expected) > 0 {
				for _, r := range deep.Equal(tt.expected, alerts) {
					t.Error(r)
				}
			}
		})
	}
}
