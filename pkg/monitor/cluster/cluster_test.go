package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

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

var errTestCollectorFailure = errors.New("failure inside")

func testCollectorNoop(ctx context.Context) error      { return nil }
func testCollectorNoopB(ctx context.Context) error     { return nil }
func testCollectorFailing(ctx context.Context) error   { return errTestCollectorFailure }
func testCollectorPanicking(ctx context.Context) error { panic(errTestCollectorFailure) }

type testCancellingCollector struct {
	cancel context.CancelFunc
}

func (c *testCancellingCollector) cancelContextCollector(ctx context.Context) error {
	c.cancel()
	return nil
}

func TestMonitor(t *testing.T) {
	var _ctx context.Context
	var _cancel context.CancelFunc

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
				return []collectorFunc{testCollectorNoop, testCollectorNoopB}
			},
			expectedGauges: []fakemetrics.MetricsAssertion[int64]{
				{
					MetricName: "apiserver.healthz.code",
					Value:      int64(1),
					Dimensions: map[string]string{
						"code": "200",
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
						"collector": "testCollectorNoop",
					},
				},
				{
					MetricName: "monitor.cluster.collector.duration",
					Value:      1.0,
					Dimensions: map[string]string{
						"collector": "testCollectorNoopB",
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
				return []collectorFunc{testCollectorFailing}
			},
			expectedErrors: []error{
				&failureToRunClusterCollector{collectorName: "testCollectorFailing"},
				errTestCollectorFailure,
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
						"collector": "testCollectorFailing",
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
				return []collectorFunc{testCollectorPanicking, testCollectorNoop}
			},
			expectedErrors: []error{
				&failureToRunClusterCollector{collectorName: "testCollectorPanicking"},
				&collectorPanic{panicValue: errTestCollectorFailure},
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
						"collector": "testCollectorPanicking",
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
						"collector": "testCollectorNoop",
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
				cc := &testCancellingCollector{cancel: _cancel}
				return []collectorFunc{cc.cancelContextCollector, testCollectorNoop}
			},
			expectedErrors: []error{
				&failureToRunClusterCollector{collectorName: "testCollectorNoop"},
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
						"collector": "testCollectorNoop",
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
						"collector": "cancelContextCollector",
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

	mon.collectors = []collectorFunc{func(ctx context.Context) error {
		t.Fatal("collector should not be called when context is already cancelled")
		return nil
	}}

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
