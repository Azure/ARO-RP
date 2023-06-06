package alertmanager_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"

	"github.com/Azure/ARO-RP/pkg/util/alertmanager"
)

func TestAlertManager_FetchPrometheusAlerts(t *testing.T) {

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		mockResponse := `[{"labels": {"alertname": "TestingAlert"}, "annotations": {"summary": "Testing summary"}}]`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer mockServer.Close()

	mockConfig := &rest.Config{}
	mockLog := logrus.NewEntry(logrus.New())

	am := alertmanager.NewAlertManager(mockConfig, mockLog)

	alertmanagerService := mockServer.URL + "/api/v2/alerts"

	alerts, err := am.FetchPrometheusAlerts(context.Background(), alertmanagerService)
	if err != nil {
		t.Fatalf("Error fetching alerts: %v", err)
	}

	expectedAlerts := []model.Alert{
		{
			Labels:      model.LabelSet{"alertname": "TestAlert"},
			Annotations: model.LabelSet{"summary": "Test summary"},
		},
	}
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
