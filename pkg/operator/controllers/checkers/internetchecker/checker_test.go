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

	operatorv1 "github.com/openshift/api/operator/v1"

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
		name          string
		responses     []*fakeResponse
		wantErr       string
		wantCondition operatorv1.ConditionStatus
	}{
		{
			name:          "200 OK",
			responses:     []*fakeResponse{okResp},
			wantCondition: operatorv1.ConditionTrue,
		},
		{
			name:          "bad request",
			responses:     []*fakeResponse{badReq},
			wantCondition: operatorv1.ConditionTrue,
		},
		{
			name:          "eventual 200 OK",
			responses:     []*fakeResponse{networkUnreach, timedoutReq, okResp},
			wantCondition: operatorv1.ConditionTrue,
		},
		{
			name:          "eventual bad request",
			responses:     []*fakeResponse{timedoutReq, networkUnreach, badReq},
			wantCondition: operatorv1.ConditionTrue,
		},
		{
			name:          "timedout request",
			responses:     []*fakeResponse{networkUnreach, timedoutReq, timedoutReq, timedoutReq, timedoutReq, timedoutReq},
			wantErr:       "https://not-used-in-test.io: context deadline exceeded",
			wantCondition: operatorv1.ConditionFalse,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			r := &checker{
				checkTimeout: 100 * time.Millisecond,
				httpClient:   &testClient{responses: test.responses},
			}
			err := r.Check([]string{urltocheck})
			utilerror.AssertErrorMessage(t, err, test.wantErr)
		})
	}
}
