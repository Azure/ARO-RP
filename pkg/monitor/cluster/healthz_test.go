package cluster

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/fake"
	fakerestclient "k8s.io/client-go/rest/fake"

	mock_discovery "github.com/Azure/ARO-RP/pkg/util/mocks/kube"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
)

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

type fakeCLI struct {
	fake.Clientset
	discovery *mock_discovery.FakeDiscoveryClient
}

func (c *fakeCLI) Discovery() discovery.DiscoveryInterface {
	return c.discovery
}

func TestEmitAPIServerStatus(t *testing.T) {
	type gauge struct {
		key  string
		code int64
		dims map[string]string
	}

	type test struct {
		name        string
		statusCode  int
		response    string
		expected    []gauge
		err         error
		errExpected bool
	}

	for _, tt := range []*test{
		{
			name:       "valid",
			statusCode: 200,
			response:   "health ok",
			expected: []gauge{
				{
					"apiserver.healthz.code",
					1,
					map[string]string{
						"code": "200",
					},
				},
				{
					"apiserver.ready",
					1,
					map[string]string{},
				},
			},
			errExpected: false,
		},
		{
			name:       "error, no subchecks",
			statusCode: 500,
			response:   "health not ok",
			expected: []gauge{
				{
					"apiserver.healthz.code",
					1,
					map[string]string{
						"code": "500",
					},
				},
				{
					"apiserver.ready",
					0,
					map[string]string{},
				},
			},
			errExpected: false,
		},
		{
			name:       "error, with subchecks",
			statusCode: 500,
			response:   "[-]test1 not ok\n[-]test2 very bad\n[+]test3 ok\nhealth not ok",
			expected: []gauge{
				{
					"apiserver.healthz.code",
					1,
					map[string]string{
						"code": "500",
					},
				},
				{
					"apiserver.ready",
					0,
					map[string]string{"failedUnit": "test1"},
				},
				{
					"apiserver.ready",
					0,
					map[string]string{"failedUnit": "test2"},
				},
			},
			errExpected: false,
		},
		{
			name:       "error, timeout",
			statusCode: 0,
			err:        errors.New("big timeout :("),
			expected: []gauge{
				{
					"apiserver.healthz.code",
					1,
					map[string]string{
						"code": "0",
					},
				},
				{
					"apiserver.ready",
					0,
					map[string]string{},
				},
			},
			errExpected: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakerestclient.RESTClient{
				Err:                  tt.err,
				NegotiatedSerializer: serializer.NegotiatedSerializerWrapper(runtime.SerializerInfo{}),
				Resp: &http.Response{
					StatusCode: tt.statusCode,
					Body:       ioutil.NopCloser(strings.NewReader(tt.response)),
				},
			}

			cli := &fakeCLI{
				discovery: &mock_discovery.FakeDiscoveryClient{
					Client: client,
				},
			}

			controller := gomock.NewController(t)
			defer controller.Finish()
			m := mock_metrics.NewMockInterface(controller)

			mon := &Monitor{
				m:   m,
				cli: cli,
			}

			for _, expected := range tt.expected {
				m.EXPECT().EmitGauge(expected.key, expected.code, expected.dims)
			}

			code, err := mon.emitAPIServerHealthzCode()

			if code != tt.statusCode {
				t.Error("unexpected error code")
			}

			if err != nil && !tt.errExpected {
				t.Error(err)
			}
		})
	}
}
