package cluster

import (
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
		errorCode   int
		response    string
		expected    []gauge
		errExpected bool
	}

	for _, tt := range []*test{
		{
			name:      "valid",
			errorCode: 200,
			response:  "bal",
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
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakerestclient.RESTClient{
				NegotiatedSerializer: serializer.NegotiatedSerializerWrapper(runtime.SerializerInfo{}),
				Resp: &http.Response{
					StatusCode: 200,
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

			if code != tt.errorCode {
				t.Error("unexpected error code")
			}

			if err != nil && !tt.errExpected {
				t.Error(err)
			}

		})
	}
}
