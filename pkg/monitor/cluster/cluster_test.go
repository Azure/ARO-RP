package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restfake "k8s.io/client-go/rest/fake"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/scheme"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
	testlog "github.com/Azure/ARO-RP/test/util/log"
	fakemetrics "github.com/Azure/ARO-RP/test/util/metrics"
)

func TestMonitor(t *testing.T) {
	var _ctx context.Context
	var _cancel context.CancelFunc

	innerFailure := errors.New("failure inside")

	for _, tt := range []struct {
		name           string
		expectedErrors []error
		hooks          func(*testclienthelper.HookingClient)
		collectors     func(*Monitor) []collectorFunc
		healthzCall    func(*http.Request) (*http.Response, error)
		expectedGauges []fakemetrics.MetricsAssertion[int64]
		expectedFloats []fakemetrics.MetricsAssertion[float64]
	}{
		{
			name:        "happy path",
			healthzCall: func(r *http.Request) (*http.Response, error) { return &http.Response{StatusCode: http.StatusOK}, nil },
			collectors: func(m *Monitor) []collectorFunc {
				return []collectorFunc{m.emitReplicasetStatuses, m.emitDaemonsetStatuses}
			},
			expectedGauges: []fakemetrics.MetricsAssertion[int64]{
				{
					MetricName: "apiserver.healthz.code",
					Value:      int64(1),
					Dimensions: map[string]string{
						"code": "200",
					},
				},
				{
					MetricName: "replicaset.statuses",
					Value:      int64(1),
					Dimensions: map[string]string{
						"availableReplicas": "1",
						"name":              "name1",
						"namespace":         "openshift",
						"replicas":          "2",
					},
				},
				{
					MetricName: "daemonset.statuses",
					Value:      int64(1),
					Dimensions: map[string]string{
						"desiredNumberScheduled": "2",
						"numberAvailable":        "1",
						"namespace":              "openshift",
						"name":                   "daemonset",
					},
				},
			},
			expectedFloats: []fakemetrics.MetricsAssertion[float64]{
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "emitAPIServerHealthzCode",
					},
				},
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "prefetchClusterVersion",
					},
				},
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "fetchManagedNamespaces",
					},
				},
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "emitReplicasetStatuses",
					},
				},
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "emitDaemonsetStatuses",
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
			expectedGauges: []fakemetrics.MetricsAssertion[int64]{
				{
					MetricName: "apiserver.healthz.code",
					Value:      int64(1),
					Dimensions: map[string]string{
						"code": "200",
					},
				},
				{
					MetricName: "monitor.cluster.collector.error",
					Value:      int64(1),
					Dimensions: map[string]string{
						"collector": "fetchManagedNamespaces",
					},
				},
			},
			expectedFloats: []fakemetrics.MetricsAssertion[float64]{
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "emitAPIServerHealthzCode",
					},
				},
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "prefetchClusterVersion",
					},
				},
			},
		},
		{
			name:        "collector failure",
			healthzCall: func(r *http.Request) (*http.Response, error) { return &http.Response{StatusCode: http.StatusOK}, nil },
			collectors: func(m *Monitor) []collectorFunc {
				return []collectorFunc{m.emitReplicasetStatuses}
			},
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
			expectedGauges: []fakemetrics.MetricsAssertion[int64]{
				{
					MetricName: "apiserver.healthz.code",
					Value:      int64(1),
					Dimensions: map[string]string{
						"code": "200",
					},
				},
				{
					MetricName: "monitor.cluster.collector.error",
					Value:      int64(1),
					Dimensions: map[string]string{
						"collector": "emitReplicasetStatuses",
					},
				},
			},
			expectedFloats: []fakemetrics.MetricsAssertion[float64]{
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "emitAPIServerHealthzCode",
					},
				},
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "prefetchClusterVersion",
					},
				},
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "fetchManagedNamespaces",
					},
				},
			},
		},
		{
			name:        "collector panic does not stop other collectors",
			healthzCall: func(r *http.Request) (*http.Response, error) { return &http.Response{StatusCode: http.StatusOK}, nil },
			collectors: func(m *Monitor) []collectorFunc {
				return []collectorFunc{m.emitReplicasetStatuses, m.emitDaemonsetStatuses}
			},
			hooks: func(hc *testclienthelper.HookingClient) {
				hc.WithPreListHook(func(obj client.ObjectList, opts *client.ListOptions) error {
					_, ok := obj.(*appsv1.ReplicaSetList)
					if ok {
						panic(innerFailure)
					}
					return nil
				})
			},
			expectedErrors: []error{
				&failureToRunClusterCollector{collectorName: "emitReplicasetStatuses"},
				&collectorPanic{panicValue: innerFailure},
			},
			expectedGauges: []fakemetrics.MetricsAssertion[int64]{
				{
					MetricName: "apiserver.healthz.code",
					Value:      int64(1),
					Dimensions: map[string]string{
						"code": "200",
					},
				},
				{
					MetricName: "daemonset.statuses",
					Value:      int64(1),
					Dimensions: map[string]string{
						"desiredNumberScheduled": "2",
						"numberAvailable":        "1",
						"namespace":              "openshift",
						"name":                   "daemonset",
					},
				},
				{
					MetricName: "monitor.cluster.collector.error",
					Value:      int64(1),
					Dimensions: map[string]string{
						"collector": "emitReplicasetStatuses",
					},
				},
			},
			expectedFloats: []fakemetrics.MetricsAssertion[float64]{
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "emitAPIServerHealthzCode",
					},
				},
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "prefetchClusterVersion",
					},
				},
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "fetchManagedNamespaces",
					},
				},
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "emitDaemonsetStatuses",
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
			expectedGauges: []fakemetrics.MetricsAssertion[int64]{
				{
					MetricName: "apiserver.healthz.code",
					Value:      int64(1),
					Dimensions: map[string]string{
						"code": "500",
					},
				},
				{
					MetricName: "monitor.cluster.collector.error",
					Value:      int64(1),
					Dimensions: map[string]string{
						"collector": "emitAPIServerHealthzCode",
					},
				},
				{
					MetricName: "monitor.cluster.collector.error",
					Value:      int64(1),
					Dimensions: map[string]string{
						"collector": "emitAPIServerPingCode",
					},
				},
				{
					MetricName: "apiserver.healthz.ping.code",
					Value:      int64(1),
					Dimensions: map[string]string{
						"code": "500",
					},
				},
			},
			expectedFloats: []fakemetrics.MetricsAssertion[float64]{},
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
			expectedGauges: []fakemetrics.MetricsAssertion[int64]{
				{
					MetricName: "apiserver.healthz.code",
					Value:      int64(1),
					Dimensions: map[string]string{
						"code": "500",
					},
				},
				{
					MetricName: "apiserver.healthz.ping.code",
					Value:      int64(1),
					Dimensions: map[string]string{
						"code": "200",
					},
				},
				{
					MetricName: "monitor.cluster.collector.error",
					Value:      int64(1),
					Dimensions: map[string]string{
						"collector": "emitAPIServerHealthzCode",
					},
				},
			},
			expectedFloats: []fakemetrics.MetricsAssertion[float64]{
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "emitAPIServerPingCode",
					},
				},
			},
		},
		{
			name:        "timeout during collector means other collectors are skipped",
			healthzCall: func(r *http.Request) (*http.Response, error) { return &http.Response{StatusCode: http.StatusOK}, nil },
			collectors: func(m *Monitor) []collectorFunc {
				return []collectorFunc{
					func(ctx context.Context) error {
						_cancel()
						return nil
					},
					func(ctx context.Context) error {
						return nil
					},
				}
			},
			expectedErrors: []error{
				&failureToRunClusterCollector{collectorName: "2"},
				context.Canceled,
			},
			expectedGauges: []fakemetrics.MetricsAssertion[int64]{
				{
					MetricName: "apiserver.healthz.code",
					Value:      int64(1),
					Dimensions: map[string]string{
						"code": "200",
					},
				},
				{
					MetricName: "monitor.cluster.collector.skipped",
					Value:      int64(1),
					Dimensions: map[string]string{
						"collector": "2",
					},
				},
			},
			expectedFloats: []fakemetrics.MetricsAssertion[float64]{
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "emitAPIServerHealthzCode",
					},
				},
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "prefetchClusterVersion",
					},
				},
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "fetchManagedNamespaces",
					},
				},
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "1",
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
				}, &appsv1.DaemonSet{ // metric expected
					ObjectMeta: metav1.ObjectMeta{
						Name:      "daemonset",
						Namespace: "openshift",
					},
					Status: appsv1.DaemonSetStatus{
						DesiredNumberScheduled: 2,
						NumberAvailable:        1,
					},
				},
			}

			_ctx, _cancel = context.WithCancel(t.Context())
			defer _cancel()

			_, log := testlog.New()
			m := fakemetrics.NewFakeMetricsEmitter(t)

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

			currTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
			now := func() time.Time {
				currTime = currTime.Add(1 * time.Second)
				return currTime
			}

			mon := &Monitor{
				log:          log,
				rawClient:    fakeRawClient,
				ocpclientset: ocpclientset,
				now:          now,
				m:            m,
				queryLimit:   1,
				parallelism:  1,
			}

			if tt.collectors != nil {
				mon.collectors = tt.collectors(mon)
			}

			err := mon.Monitor(_ctx)
			utilerror.AssertErrorMatchesAll(t, err, tt.expectedErrors)

			// we only emit duration when no errors
			f := tt.expectedFloats
			if len(tt.expectedErrors) == 0 {
				f = append(tt.expectedFloats, fakemetrics.MetricsAssertion[float64]{
					MetricName: "monitor.cluster.duration",
					Value:      currTime.Sub(time.Date(1970, 1, 1, 0, 0, 1, 0, time.UTC)).Seconds(),
					Dimensions: map[string]string{},
				})
			}

			m.AssertFloats(f...)
			m.AssertGauges(tt.expectedGauges...)
		})
	}
}

func TestMonitorAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	_, log := testlog.New()
	m := fakemetrics.NewFakeMetricsEmitter(t)

	// for healthz
	fakeRawClient := &restfake.RESTClient{
		NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		Client: restfake.CreateHTTPClient(
			func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK}, nil
			}),
	}

	client := testclienthelper.NewHookingClient(fake.
		NewClientBuilder().
		Build())
	ocpclientset := clienthelper.NewWithClient(log, client)

	currTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	now := func() time.Time {
		currTime = currTime.Add(1 * time.Second)
		return currTime
	}

	mon := &Monitor{
		log:          log,
		rawClient:    fakeRawClient,
		ocpclientset: ocpclientset,
		m:            m,
		queryLimit:   1,
		now:          now,
	}

	mon.collectors = []collectorFunc{mon.emitDaemonsetStatuses}

	// Cancel context before it hits the monitor
	cancel()

	err := mon.Monitor(ctx)
	utilerror.AssertErrorMatchesAll(t, err, []error{
		&failureToRunClusterCollector{collectorName: "emitAPIServerHealthzCode"},
		&failureToRunClusterCollector{collectorName: "emitAPIServerPingCode"},
		context.Canceled,
	})

	m.AssertFloats()
	m.AssertGauges([]fakemetrics.MetricsAssertion[int64]{
		{
			MetricName: "monitor.cluster.collector.skipped",
			Value:      1,
			Dimensions: map[string]string{
				"collector": "emitAPIServerPingCode",
			},
		},
		{
			MetricName: "monitor.cluster.collector.skipped",
			Value:      1,
			Dimensions: map[string]string{
				"collector": "emitAPIServerHealthzCode",
			},
		},
	}...)
}
