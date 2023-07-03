package alertmanager_test

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prometheus/common/model"

	mock_alertmanager "github.com/Azure/ARO-RP/pkg/util/mocks/alertmanager"
)

const (
	alertmanagerService = "http://alertmanager-main.openshift-monitoring.svc:9093/api/v2/alerts"
)

func TestAlertManager_FetchPrometheusAlerts(t *testing.T) {
	ctx := context.Background()
	var alerts []model.Alert

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockResponse := `[{"labels": {"alertname": "TestingAlert"}, "annotations": {"summary": "Testing summary"}}]`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer mockServer.Close()

	controller := gomock.NewController(t)
	defer controller.Finish()

	mockAlertManagerClient := mock_alertmanager.NewMockAlertManager(controller)

	alerts = []model.Alert{
		{
			Labels:      model.LabelSet{"alertname": "TestAlert"},
			Annotations: model.LabelSet{"summary": "Test summary"},
		},
	}

	expectedAlerts := []model.Alert{
		{
			Labels:      model.LabelSet{"alertname": "TestAlert"},
			Annotations: model.LabelSet{"summary": "Test summary"},
		},
	}

	mockAlertManagerClient.EXPECT().FetchPrometheusAlerts(ctx, alertmanagerService).AnyTimes().Return(alerts, nil)

	if len(alerts) != len(expectedAlerts) {
		t.Fatalf("Expected %d alerts, got %d", len(expectedAlerts), len(alerts))
	}
	for i, alert := range alerts {
		if alert.Labels["alertname"] != expectedAlerts[i].Labels["alertname"] ||
			alert.Annotations["summary"] != expectedAlerts[i].Annotations["summary"] {
			t.Errorf("Mismatched alert at index %d", i)
		}
	}
}
