package internetchecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	operatorv1 "github.com/openshift/api/operator/v1"

	"github.com/Azure/ARO-RP/pkg/operator"
	mock_metrics "github.com/Azure/ARO-RP/pkg/util/mocks/operator/metrics"
	utilerror "github.com/Azure/ARO-RP/test/util/error"
)

type fakeResponse struct {
	httpResponse *http.Response
	err          error
}

type testClient struct {
	responses []*fakeResponse
}

func (c *testClient) Do(req *http.Request) (*http.Response, error) {
	response := c.responses[0]
	c.responses = c.responses[1:]
	return response.httpResponse, response.err
}

const urltocheck = "https://not-used-in-test.io"

// simulated responses
var (
	okResp = &fakeResponse{
		httpResponse: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(nil),
		},
	}

	badReq = &fakeResponse{
		httpResponse: &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(nil),
		},
	}

	networkUnreach = &fakeResponse{
		err: &url.Error{
			URL: urltocheck,
			Err: &net.OpError{
				Err: os.NewSyscallError("socket", syscall.ENETUNREACH),
			},
		},
	}

	timedoutReq = &fakeResponse{err: context.DeadlineExceeded}
)

func TestCheck(t *testing.T) {
	var testCases = []struct {
		name            string
		responses       []*fakeResponse
		wantErr         string
		wantCondition   operatorv1.ConditionStatus
		wantMetricValue bool
	}{
		{
			name:            "200 OK",
			responses:       []*fakeResponse{okResp},
			wantCondition:   operatorv1.ConditionTrue,
			wantMetricValue: true,
		},
		{
			name:            "bad request",
			responses:       []*fakeResponse{badReq},
			wantCondition:   operatorv1.ConditionTrue,
			wantMetricValue: true,
		},
		{
			name:            "eventual 200 OK",
			responses:       []*fakeResponse{networkUnreach, timedoutReq, okResp},
			wantCondition:   operatorv1.ConditionTrue,
			wantMetricValue: true,
		},
		{
			name:            "eventual bad request",
			responses:       []*fakeResponse{timedoutReq, networkUnreach, badReq},
			wantCondition:   operatorv1.ConditionTrue,
			wantMetricValue: true,
		},
		{
			name:            "timedout request",
			responses:       []*fakeResponse{networkUnreach, timedoutReq, timedoutReq, timedoutReq, timedoutReq, timedoutReq},
			wantErr:         "https://not-used-in-test.io: context deadline exceeded",
			wantCondition:   operatorv1.ConditionFalse,
			wantMetricValue: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			metricsClientFake := mock_metrics.NewMockClient(controller)
			metricsClientFake.EXPECT().UpdateRequiredEndpointAccessible(urltocheck, operator.RoleMaster, tt.wantMetricValue)

			r := &checker{
				checkTimeout:  100 * time.Millisecond,
				httpClient:    &testClient{responses: tt.responses},
				metricsClient: metricsClientFake,
				role:          operator.RoleMaster,
			}
			err := r.Check([]string{urltocheck})
			utilerror.AssertErrorMessage(t, err, tt.wantErr)
		})
	}
}
