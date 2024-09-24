package cosmosdb

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/metrics"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

type testRoundTripper struct {
	resp *http.Response
	err  error
}

func (rt *testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return rt.resp, rt.err
}

func TestTracerRoundTripperRoundTrip(t *testing.T) {
	for _, tt := range []struct {
		name               string
		url                string
		rt                 http.RoundTripper
		mocks              func(*mock_metrics.MockEmitter)
		wantErr            string
		wantRespStatusCode int
	}{
		{
			name: "rt returns an error",
			rt: &testRoundTripper{
				err: errors.New("roundtrip failed"),
			},
			mocks: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().EmitGauge("client.cosmosdb.count", int64(1), map[string]string{
					"verb": http.MethodGet,
					"path": "/foo",
					"code": "0",
				})
				m.EXPECT().EmitGauge("client.cosmosdb.duration", gomock.Any(), map[string]string{
					"verb": http.MethodGet,
					"path": "/foo",
					"code": "0",
				})
				m.EXPECT().EmitGauge("client.cosmosdb.errors", int64(1), map[string]string{
					"verb": http.MethodGet,
					"path": "/foo",
					"code": "0",
				})
			},
			wantErr: "roundtrip failed",
		},
		{
			name: "invalid request charge",
			rt: &testRoundTripper{
				resp: &http.Response{
					StatusCode: http.StatusUnauthorized,
					Header: http.Header{
						"X-Ms-Request-Charge": {`"hello"`},
					},
				},
			},
			mocks: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().EmitGauge("client.cosmosdb.count", int64(1), map[string]string{
					"verb": http.MethodGet,
					"path": "/foo",
					"code": "401",
				})
				m.EXPECT().EmitGauge("client.cosmosdb.duration", gomock.Any(), map[string]string{
					"verb": http.MethodGet,
					"path": "/foo",
					"code": "401",
				})
			},
			wantRespStatusCode: http.StatusUnauthorized,
		},
		{
			name: "valid request charge with docs URL as well",
			url:  "http://example.com/docs/random-id",
			rt: &testRoundTripper{
				resp: &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"X-Ms-Request-Charge": {`"1.23"`},
					},
				},
			},
			mocks: func(m *mock_metrics.MockEmitter) {
				m.EXPECT().EmitGauge("client.cosmosdb.count", int64(1), map[string]string{
					"verb": http.MethodGet,
					"path": "/docs/{id}",
					"code": "200",
				})
				m.EXPECT().EmitGauge("client.cosmosdb.duration", gomock.Any(), map[string]string{
					"verb": http.MethodGet,
					"path": "/docs/{id}",
					"code": "200",
				})
				m.EXPECT().EmitFloat("client.cosmosdb.requestunits", 1.23, map[string]string{
					"verb": http.MethodGet,
					"path": "/docs/{id}",
					"code": "200",
				})
			},
			wantRespStatusCode: http.StatusOK,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			m := mock_metrics.NewMockEmitter(controller)
			tt.mocks(m)

			tripper := &tracerRoundTripper{
				log: logrus.NewEntry(logrus.StandardLogger()),
				m:   m,
				tr:  tt.rt,
			}

			url := "http://example.com/foo"
			if tt.url != "" {
				url = tt.url
			}

			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Fatal(err)
			}

			resp, err := tripper.RoundTrip(req)
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
			if resp != nil && resp.StatusCode != tt.wantRespStatusCode ||
				resp == nil && tt.wantRespStatusCode != 0 {
				t.Error(resp)
			}
		})
	}
}
