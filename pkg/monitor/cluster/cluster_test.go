package cluster

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	restfake "k8s.io/client-go/rest/fake"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	configv1 "github.com/openshift/api/config/v1"

	"github.com/Azure/ARO-RP/pkg/operator/clientset/versioned/scheme"
	"github.com/Azure/ARO-RP/pkg/util/clienthelper"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	testclienthelper "github.com/Azure/ARO-RP/test/util/clienthelper"
	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestMonitor(t *testing.T) {
	ctx := context.Background()

	for _, tt := range []struct {
		name           string
		expectedErrors []error
		hooks          func(*testclienthelper.HookingClient)
		healthzCall    func(*http.Request) (*http.Response, error)
		expectedGauges []struct {
			name   string
			value  int64
			labels map[string]string
		}
		expectedFloats []struct {
			name   string
			value  any
			labels map[string]string
		}
	}{
		{
			name:        "happy path",
			healthzCall: func(r *http.Request) (*http.Response, error) { return &http.Response{StatusCode: http.StatusOK}, nil },
			expectedGauges: []struct {
				name   string
				value  int64
				labels map[string]string
			}{
				{
					name:  "apiserver.healthz.code",
					value: 1,
					labels: map[string]string{
						"code": "200",
					},
				},
				{
					name:  "replicaset.statuses",
					value: 1,
					labels: map[string]string{
						"availableReplicas": "1",
						"name":              "name1",
						"namespace":         "openshift",
						"replicas":          "2",
					},
				},
			},
			expectedFloats: []struct {
				name   string
				value  any
				labels map[string]string
			}{
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
				fmt.Errorf("failure running cluster collector 'fetchManagedNamespaces': %w", fmt.Errorf("error in list operation: %w", errors.New("failure with ns"))),
			},
			expectedGauges: []struct {
				name   string
				value  int64
				labels map[string]string
			}{
				{
					name:  "apiserver.healthz.code",
					value: 1,
					labels: map[string]string{
						"code": "200",
					},
				},
				{
					name:  "monitor.cluster.collector.error",
					value: 1,
					labels: map[string]string{
						"collector": "fetchManagedNamespaces",
					},
				},
			},
			expectedFloats: []struct {
				name   string
				value  any
				labels map[string]string
			}{
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
						return errors.New("failure with replicaset")
					}
					return nil
				})
			},
			expectedErrors: []error{
				fmt.Errorf("failure running cluster collector 'emitReplicasetStatuses': %w", fmt.Errorf("error in list operation: %w", errors.New("failure with replicaset"))),
			},
			expectedGauges: []struct {
				name   string
				value  int64
				labels map[string]string
			}{
				{
					name:  "apiserver.healthz.code",
					value: 1,
					labels: map[string]string{
						"code": "200",
					},
				},
				{
					name:  "monitor.cluster.collector.error",
					value: 1,
					labels: map[string]string{
						"collector": "emitReplicasetStatuses",
					},
				},
			},
			expectedFloats: []struct {
				name   string
				value  any
				labels map[string]string
			}{
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
				fmt.Errorf("failure running cluster collector 'emitAPIServerHealthzCode': %w", kerrors.NewGenericServerResponse(500, "GET", schema.GroupResource{}, "", "", 0, true)),
				fmt.Errorf("failure running cluster collector 'emitAPIServerPingCode': %w", kerrors.NewGenericServerResponse(500, "GET", schema.GroupResource{}, "", "", 0, true)),
			},
			expectedGauges: []struct {
				name   string
				value  int64
				labels map[string]string
			}{
				{
					name:  "apiserver.healthz.code",
					value: 1,
					labels: map[string]string{
						"code": "500",
					},
				},
				{
					name:  "monitor.cluster.collector.error",
					value: 1,
					labels: map[string]string{
						"collector": "emitAPIServerHealthzCode",
					},
				},
				{
					name:  "monitor.cluster.collector.error",
					value: 1,
					labels: map[string]string{
						"collector": "emitAPIServerPingCode",
					},
				},
				{
					name:  "apiserver.healthz.ping.code",
					value: 1,
					labels: map[string]string{
						"code": "500",
					},
				},
			},
			expectedFloats: []struct {
				name   string
				value  any
				labels map[string]string
			}{},
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
				fmt.Errorf("failure running cluster collector 'emitAPIServerHealthzCode': %w", kerrors.NewGenericServerResponse(500, "GET", schema.GroupResource{}, "", "", 0, true)),
			},
			expectedGauges: []struct {
				name   string
				value  int64
				labels map[string]string
			}{
				{
					name:  "apiserver.healthz.code",
					value: 1,
					labels: map[string]string{
						"code": "500",
					},
				},
				{
					name:  "apiserver.healthz.ping.code",
					value: 1,
					labels: map[string]string{
						"code": "200",
					},
				},
				{
					name:  "monitor.cluster.collector.error",
					value: 1,
					labels: map[string]string{
						"collector": "emitAPIServerHealthzCode",
					},
				},
			},
			expectedFloats: []struct {
				name   string
				value  any
				labels map[string]string
			}{
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

			outerWg := new(sync.WaitGroup)
			outerWg.Add(1)

			mon := &Monitor{
				log:          log,
				rawClient:    fakeRawClient,
				ocpclientset: ocpclientset,
				m:            m,
				queryLimit:   1,
				wg:           outerWg,
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

			errs := mon.Monitor(ctx)
			assert.Equal(t, tt.expectedErrors, errs)
		})
	}
}
