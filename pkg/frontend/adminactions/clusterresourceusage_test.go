package adminactions

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake "k8s.io/client-go/kubernetes/fake"
	restclient "k8s.io/client-go/rest"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

// fakeKubeActions wraps a fake Kubernetes client.
type fakeKubeActions struct {
	kubecli *fake.Clientset
}

func (fka *fakeKubeActions) TopPods(ctx context.Context, restConfig *restclient.Config, allNamespaces bool) ([]PodMetrics, error) {
	return (&kubeActions{kubecli: fka.kubecli}).TopPods(ctx, restConfig, allNamespaces)
}

func (fka *fakeKubeActions) TopNodes(ctx context.Context, restConfig *restclient.Config) ([]NodeMetrics, error) {
	return (&kubeActions{kubecli: fka.kubecli}).TopNodes(ctx, restConfig)
}

func TestCalculatePercentage(t *testing.T) {
	res := calculatePercentage("100m", 1000)
	expected := 10.0
	if res != expected {
		t.Errorf("calculatePercentage(\"100m\", 1000) = %f; want %f", res, expected)
	}
}

func TestRoundPercentage(t *testing.T) {
	res := roundPercentage(12.3456)
	expected := 12.35
	if res != expected {
		t.Errorf("roundPercentage(12.3456) = %f; want %f", res, expected)
	}
	if roundPercentage(0) != 0 {
		t.Error("roundPercentage(0) should be 0")
	}
}

func TestTopPodsTable(t *testing.T) {
	type topPodsTestCase struct {
		name             string
		allNamespaces    bool
		restConfig       *restclient.Config // if provided, overrides server setup
		setupServer      func() *httptest.Server
		expectedError    string
		expectedPodCount int
	}

	testCases := []topPodsTestCase{
		{
			name:          "Valid Input",
			allNamespaces: true,
			setupServer: func() *httptest.Server {
				podMetricsList := metricsv1beta1.PodMetricsList{
					Items: []metricsv1beta1.PodMetrics{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "pod-sample",
								Namespace: "openshift-test",
							},
							Containers: []metricsv1beta1.ContainerMetrics{
								{
									Name: "container-1",
									Usage: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("100m"),
										corev1.ResourceMemory: resource.MustParse("200Mi"),
									},
								},
							},
						},
					},
				}
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.HasPrefix(r.URL.Path, "/apis/metrics.k8s.io/v1beta1/pods") {
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(podMetricsList)
						return
					}
					http.NotFound(w, r)
				}))
			},
			expectedError:    "",
			expectedPodCount: 1,
		},
		{
			name:             "Missing Namespace Parameter",
			allNamespaces:    false,
			setupServer:      nil,
			expectedError:    "explicit namespace must be provided when allNamespaces is false",
			expectedPodCount: 0,
		},
		{
			name:          "Client Initialization Failure",
			allNamespaces: true,
			restConfig:    &restclient.Config{Host: ""},
			setupServer:   nil,
			expectedError: "connect: connection refused",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var restConfig *restclient.Config
			var ts *httptest.Server
			if tc.restConfig != nil {
				restConfig = tc.restConfig
			} else if tc.setupServer != nil {
				ts = tc.setupServer()
				defer ts.Close()
				restConfig = &restclient.Config{Host: ts.URL}
			} else {
				restConfig = &restclient.Config{Host: "http://invalid"}
			}

			kubeClient := fake.NewSimpleClientset(
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod-sample",
						Namespace: "openshift-test",
					},
					Spec: corev1.PodSpec{NodeName: "node-1"},
				},
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
					Status: corev1.NodeStatus{
						Capacity: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("4"),
							corev1.ResourceMemory: resource.MustParse("8Gi"),
						},
					},
				},
			)

			fka := &fakeKubeActions{kubecli: kubeClient}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			pods, err := fka.TopPods(ctx, restConfig, tc.allNamespaces)
			if tc.expectedError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.expectedError) {
					t.Fatalf("expected error containing %q, got: %v", tc.expectedError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(pods) != tc.expectedPodCount {
				t.Fatalf("expected %d pods, got %d", tc.expectedPodCount, len(pods))
			}
		})
	}
}

func TestTopNodesTable(t *testing.T) {
	type topNodesTestCase struct {
		name              string
		restConfig        *restclient.Config
		setupServer       func() *httptest.Server
		expectedError     string
		expectedNodeCount int
	}

	testCases := []topNodesTestCase{
		{
			name: "Valid Input",
			setupServer: func() *httptest.Server {
				nodeMetricsList := metricsv1beta1.NodeMetricsList{
					Items: []metricsv1beta1.NodeMetrics{
						{
							ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
							Usage: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("500m"),
								corev1.ResourceMemory: resource.MustParse("1Gi"),
							},
						},
					},
				}
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.HasPrefix(r.URL.Path, "/apis/metrics.k8s.io/v1beta1/nodes") {
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(nodeMetricsList)
						return
					}
					http.NotFound(w, r)
				}))
			},
			expectedError:     "",
			expectedNodeCount: 1,
		},
		{
			name:              "Client Initialization Failure",
			restConfig:        &restclient.Config{Host: ""},
			setupServer:       nil,
			expectedError:     "connect: connection refused",
			expectedNodeCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var restConfig *restclient.Config
			var ts *httptest.Server
			if tc.restConfig != nil {
				restConfig = tc.restConfig
			} else if tc.setupServer != nil {
				ts = tc.setupServer()
				defer ts.Close()
				restConfig = &restclient.Config{Host: ts.URL}
			} else {
				restConfig = &restclient.Config{Host: "http://invalid"}
			}

			kubeClient := fake.NewSimpleClientset(
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
					Status: corev1.NodeStatus{
						Capacity: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("4"),
							corev1.ResourceMemory: resource.MustParse("8Gi"),
						},
					},
				},
			)
			fka := &fakeKubeActions{kubecli: kubeClient}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			nodes, err := fka.TopNodes(ctx, restConfig)
			if tc.expectedError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.expectedError) {
					t.Fatalf("expected error containing %q, got: %v", tc.expectedError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(nodes) != tc.expectedNodeCount {
				t.Fatalf("expected %d nodes, got %d", tc.expectedNodeCount, len(nodes))
			}
		})
	}
}
