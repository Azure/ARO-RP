package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"k8s.io/client-go/rest"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

func TestQueryPrometheus(t *testing.T) {
	for _, tt := range []struct {
		name           string
		serverResponse string
		serverStatus   int
		query          string
		expectError    bool
		expectResults  int
	}{
		{
			name:         "successful query with results",
			serverStatus: http.StatusOK,
			serverResponse: `{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{
							"metric": {"__name__": "kubevirt_vmi_info", "name": "test-vmi"},
							"value": [1435781451.781, "1"]
						}
					]
				}
			}`,
			query:         `{__name__="kubevirt_vmi_info"}`,
			expectError:   false,
			expectResults: 1,
		},
		{
			name:         "successful query with multiple results",
			serverStatus: http.StatusOK,
			serverResponse: `{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{
							"metric": {"__name__": "up", "job": "prometheus"},
							"value": [1435781451.781, "1"]
						},
						{
							"metric": {"__name__": "up", "job": "node"},
							"value": [1435781451.781, "0"]
						}
					]
				}
			}`,
			query:         `up`,
			expectError:   false,
			expectResults: 2,
		},
		{
			name:         "successful query with no results",
			serverStatus: http.StatusOK,
			serverResponse: `{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": []
				}
			}`,
			query:         `{__name__="nonexistent"}`,
			expectError:   false,
			expectResults: 0,
		},
		{
			name:         "query returns error status",
			serverStatus: http.StatusOK,
			serverResponse: `{
				"status": "error",
				"data": {
					"resultType": "vector",
					"result": []
				}
			}`,
			query:       `invalid query`,
			expectError: true,
		},
		{
			name:           "server returns non-200 status",
			serverStatus:   http.StatusInternalServerError,
			serverResponse: `Internal Server Error`,
			query:          `{__name__="kubevirt_vmi_info"}`,
			expectError:    true,
		},
		{
			name:           "server returns invalid JSON",
			serverStatus:   http.StatusOK,
			serverResponse: `invalid json`,
			query:          `{__name__="kubevirt_vmi_info"}`,
			expectError:    true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			pm := &prometheusMetrics{
				mon: &Monitor{
					restconfig: &rest.Config{
						BearerToken: "test-token",
					},
					log: logrus.NewEntry(logrus.New()),
				},
				client: &http.Client{
					Transport: &http.Transport{},
				},
				prometheusQueryURL: server.URL + "/api/v1/query?query=%s",
			}

			results, err := pm.queryPrometheus(ctx, tt.query)

			if tt.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(results) != tt.expectResults {
				t.Fatalf("expected %d results, got %d", tt.expectResults, len(results))
			}

			if !tt.expectError && tt.expectResults > 0 {
				for _, result := range results {
					if result.Metric == nil {
						t.Fatal("expected metric to be non-nil")
					}
					if result.Value == nil {
						t.Fatal("expected value to be non-nil")
					}
					if len(result.Value) != 2 {
						t.Fatalf("expected value to have 2 elements, got %d", len(result.Value))
					}
				}
			}
		})
	}
}

func TestEmitCNVMetrics(t *testing.T) {
	for _, tt := range []struct {
		name             string
		serverResponse   string
		serverStatus     int
		expectError      bool
		expectedMetrics  int
		expectMetricDims []map[string]string
	}{
		{
			name:         "emits metrics for VMIs",
			serverStatus: http.StatusOK,
			serverResponse: `{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{
							"metric": {"__name__": "kubevirt_vmi_info", "name": "test-vmi-1", "namespace": "test-ns"},
							"value": [1435781451.781, "1"]
						},
						{
							"metric": {"__name__": "kubevirt_vmi_info", "name": "test-vmi-2", "namespace": "test-ns"},
							"value": [1435781451.781, "1"]
						}
					]
				}
			}`,
			expectError:     false,
			expectedMetrics: 2,
			expectMetricDims: []map[string]string{
				{"__name__": "kubevirt_vmi_info", "name": "test-vmi-1", "namespace": "test-ns"},
				{"__name__": "kubevirt_vmi_info", "name": "test-vmi-2", "namespace": "test-ns"},
			},
		},
		{
			name:         "handles empty results",
			serverStatus: http.StatusOK,
			serverResponse: `{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": []
				}
			}`,
			expectError:     false,
			expectedMetrics: 0,
		},
		{
			name:            "handles query error",
			serverStatus:    http.StatusInternalServerError,
			serverResponse:  `Internal Server Error`,
			expectError:     true,
			expectedMetrics: 0,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			ctx := context.Background()
			m := mock_metrics.NewMockEmitter(controller)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			mon := &Monitor{
				m:    m,
				dims: map[string]string{},
				restconfig: &rest.Config{
					BearerToken: "test-token",
				},
				log: logrus.NewEntry(logrus.New()),
			}

			pm := &prometheusMetrics{
				mon: mon,
				client: &http.Client{
					Transport: &http.Transport{},
				},
				prometheusQueryURL: server.URL + "/api/v1/query?query=%s",
			}

			for _, dims := range tt.expectMetricDims {
				m.EXPECT().EmitGauge("cnv.kubevirt.vmi.info", int64(1), dims)
			}

			err := pm.emitCNVMetrics(ctx)

			if tt.expectError && err == nil {
				t.Fatal("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
