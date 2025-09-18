package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"go.uber.org/mock/gomock"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restfake "k8s.io/client-go/rest/fake"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/scheme"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

type expectedMetric struct {
	name   string
	value  any
	labels map[string]string
}

func TestMonitor(t *testing.T) {
	ctx := context.Background()

	innerFailure := errors.New("failure inside")

	for _, tt := range []struct {
		name           string
		expectedErrors []error
		hooks          func(*testclienthelper.HookingClient)
		healthzCall    func(*http.Request) (*http.Response, error)
		expectedGauges []expectedMetric
		expectedFloats []expectedMetric
	}{
		{
			name:        "happy path",
			healthzCall: func(r *http.Request) (*http.Response, error) { return &http.Response{StatusCode: http.StatusOK}, nil },
			expectedGauges: []expectedMetric{
				{
					name:  "apiserver.healthz.code",
					value: int64(1),
					labels: map[string]string{
						"code": "200",
					},
				},
				{
					name:  "replicaset.statuses",
					value: int64(1),
					labels: map[string]string{
						"availableReplicas": "1",
						"name":              "name1",
						"namespace":         "openshift",
						"replicas":          "2",
					},
				},
			},
			expectedFloats: []expectedMetric{
				{
					name:  "monitor.cluster.collector.duration",
					value: gomock.Any(),
					labels: map[string]string{
						"collector": "emitAPIServerHealthzCode",
					},
				},
				{
					name:  "monitor.cluster.collector.duration",
					value: gomock.Any(),
					labels: map[string]string{
						"collector": "emitReplicasetStatuses",
					},
				},
				{
					name:  "monitor.cluster.collector.duration",
					value: gomock.Any(),
					labels: map[string]string{
						"collector": "prefetchClusterVersion",
					},
				},
				{
					name:  "monitor.cluster.collector.duration",
					value: gomock.Any(),
					labels: map[string]string{
						"collector": "fetchManagedNamespaces",
					},
				},
			},
		},
		{
			name:        "namespace fetch failure",
			healthzCall: func(r *http.Request) (*http.Response, error) { return &http.Response{StatusCode: http.StatusOK}, nil },
			hooks: func(hc *testclienthelper.HookingClient) {
				hc.WithPreListHook(func(obj client.ObjectList, opts *client.ListOptions) error {
					_, ok := obj.(*corev1.NamespaceList)
					if ok {
						return errors.New("failure with ns")
					}
					return nil
				})
			},
			expectedErrors: []error{
				&failureToRunClusterCollector{collectorName: "fetchManagedNamespaces"},
				errListNamespaces,
			},
			expectedGauges: []expectedMetric{
				{
					name:  "apiserver.healthz.code",
					value: int64(1),
					labels: map[string]string{
						"code": "200",
					},
				},
				{
					name:  "monitor.cluster.collector.error",
					value: int64(1),
					labels: map[string]string{
						"collector": "fetchManagedNamespaces",
					},
				},
			},
			expectedFloats: []expectedMetric{
				{
					name:  "monitor.cluster.collector.duration",
					value: gomock.Any(),
					labels: map[string]string{
						"collector": "emitAPIServerHealthzCode",
					},
				},
				{
					name:  "monitor.cluster.collector.duration",
					value: gomock.Any(),
					labels: map[string]string{
						"collector": "prefetchClusterVersion",
					},
				},
			},
		},
		{
			name:        "collector failure",
			healthzCall: func(r *http.Request) (*http.Response, error) { return &http.Response{StatusCode: http.StatusOK}, nil },
			hooks: func(hc *testclienthelper.HookingClient) {
				hc.WithPreListHook(func(obj client.ObjectList, opts *client.ListOptions) error {
					_, ok := obj.(*appsv1.ReplicaSetList)
					if ok {
						return innerFailure
					}
					return nil
				})
			},
			expectedErrors: []error{
				&failureToRunClusterCollector{collectorName: "emitReplicasetStatuses"},
				errListReplicaSets,
				innerFailure,
			},
			expectedGauges: []expectedMetric{
				{
					name:  "apiserver.healthz.code",
					value: int64(1),
					labels: map[string]string{
						"code": "200",
					},
				},
				{
					name:  "monitor.cluster.collector.error",
					value: int64(1),
					labels: map[string]string{
						"collector": "emitReplicasetStatuses",
					},
				},
			},
			expectedFloats: []expectedMetric{
				{
					name:  "monitor.cluster.collector.duration",
					value: gomock.Any(),
					labels: map[string]string{
						"collector": "emitAPIServerHealthzCode",
					},
				},
				{
					name:  "monitor.cluster.collector.duration",
					value: gomock.Any(),
					labels: map[string]string{
						"collector": "prefetchClusterVersion",
					},
				},
				{
					name:  "monitor.cluster.collector.duration",
					value: gomock.Any(),
					labels: map[string]string{
						"collector": "fetchManagedNamespaces",
					},
				},
			},
		},
		{
			name: "both healthz failures",
			healthzCall: func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusInternalServerError}, nil
			},
			expectedErrors: []error{
				errAPIServerHealthzFailure,
				errAPIServerPingFailure,
			},
			expectedGauges: []expectedMetric{
				{
					name:  "apiserver.healthz.code",
					value: int64(1),
					labels: map[string]string{
						"code": "500",
					},
				},
				{
					name:  "monitor.cluster.collector.error",
					value: int64(1),
					labels: map[string]string{
						"collector": "emitAPIServerHealthzCode",
					},
				},
				{
					name:  "monitor.cluster.collector.error",
					value: int64(1),
					labels: map[string]string{
						"collector": "emitAPIServerPingCode",
					},
				},
				{
					name:  "apiserver.healthz.ping.code",
					value: int64(1),
					labels: map[string]string{
						"code": "500",
					},
				},
			},
			expectedFloats: []expectedMetric{},
		},
		{
			name: "api failure, ping succeeds",
			healthzCall: func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/healthz/ping" {
					return &http.Response{StatusCode: http.StatusOK}, nil
				}
				return &http.Response{StatusCode: http.StatusInternalServerError}, nil
			},
			expectedErrors: []error{
				errAPIServerHealthzFailure,
			},
			expectedGauges: []expectedMetric{
				{
					name:  "apiserver.healthz.code",
					value: int64(1),
					labels: map[string]string{
						"code": "500",
					},
				},
				{
					name:  "apiserver.healthz.ping.code",
					value: int64(1),
					labels: map[string]string{
						"code": "200",
					},
				},
				{
					name:  "monitor.cluster.collector.error",
					value: int64(1),
					labels: map[string]string{
						"collector": "emitAPIServerHealthzCode",
					},
				},
			},
			expectedFloats: []expectedMetric{
				{
					name:  "monitor.cluster.collector.duration",
					value: gomock.Any(),
					labels: map[string]string{
						"collector": "emitAPIServerPingCode",
					},
				},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			objects := []client.Object{
				namespaceObject("openshift"),
				namespaceObject("customer"),
				&configv1.ClusterVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name: "version",
					},
					Status: configv1.ClusterVersionStatus{
						History: []configv1.UpdateHistory{
							{
								State:   configv1.CompletedUpdate,
								Version: "4.16.1",
							},
						},
					},
				},
				&appsv1.ReplicaSet{ // metrics expected
					ObjectMeta: metav1.ObjectMeta{
						Name:      "name1",
						Namespace: "openshift",
					},
					Status: appsv1.ReplicaSetStatus{
						Replicas:          2,
						AvailableReplicas: 1,
					},
				}, &appsv1.ReplicaSet{ // no metric expected
					ObjectMeta: metav1.ObjectMeta{
						Name:      "name2",
						Namespace: "openshift",
					},
					Status: appsv1.ReplicaSetStatus{
						Replicas:          2,
						AvailableReplicas: 2,
					},
				}, &appsv1.ReplicaSet{
					ObjectMeta: metav1.ObjectMeta{ // no metric expected -customer
						Name:      "name2",
						Namespace: "customer",
					},
					Status: appsv1.ReplicaSetStatus{
						Replicas:          2,
						AvailableReplicas: 1,
					},
				},
			}

			_, log := testlog.New()
			controller := gomock.NewController(t)
			m := mock_metrics.NewMockEmitter(controller)

			// for healthz
			fakeRawClient := &restfake.RESTClient{
				NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
				Client:               restfake.CreateHTTPClient(tt.healthzCall),
			}

			client := testclienthelper.NewHookingClient(fake.
				NewClientBuilder().
				WithObjects(objects...).
				Build())
			ocpclientset := clienthelper.NewWithClient(log, client)

			if tt.hooks != nil {
				tt.hooks(client)
			}

			mon := &Monitor{
				log:          log,
				rawClient:    fakeRawClient,
				ocpclientset: ocpclientset,
				m:            m,
				queryLimit:   1,
			}

			mon.collectors = []func(context.Context) error{
				mon.emitReplicasetStatuses,
			}

			for _, gauge := range tt.expectedGauges {
				m.EXPECT().EmitGauge(gauge.name, gauge.value, gauge.labels).Times(1)
			}
			for _, gauge := range tt.expectedFloats {
				m.EXPECT().EmitFloat(gauge.name, gauge.value, gauge.labels).Times(1)
			}

			// we only emit duration when no errors
			if len(tt.expectedErrors) == 0 {
				m.EXPECT().EmitFloat("monitor.cluster.duration", gomock.Any(), gomock.Any()).Times(1)
			}

			err := mon.Monitor(ctx)
			utilerror.AssertErrorMatchesAll(t, err, tt.expectedErrors)
		})
	}
}
