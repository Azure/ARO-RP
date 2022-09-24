package checker

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

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
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

type testCase struct {
	name      string
	responses []*fakeResponse
	wantError bool
}

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

var testCases = []testCase{
	{
		name:      "200 OK",
		responses: []*fakeResponse{okResp},
	},
	{
		name:      "bad request",
		responses: []*fakeResponse{badReq},
	},
	{
		name:      "eventual 200 OK",
		responses: []*fakeResponse{networkUnreach, timedoutReq, okResp},
	},
	{
		name:      "eventual bad request",
		responses: []*fakeResponse{timedoutReq, networkUnreach, badReq},
	},
	{
		name:      "timedout request",
		responses: []*fakeResponse{networkUnreach, timedoutReq, timedoutReq, timedoutReq, timedoutReq, timedoutReq},
		wantError: true,
	},
}

func TestInternetCheckerCheck(t *testing.T) {
	r := &InternetChecker{log: utillog.GetLogger()}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client := &testClient{responses: test.responses}
			err := r.checkWithRetry(client, urltocheck, 100*time.Millisecond)
			if (err != nil) != test.wantError {
				t.Errorf("InternetChecker.check() error = %v, wantErr %v", err, test.wantError)
			}
		})
	}
}
