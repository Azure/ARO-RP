package internetchecker

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"bytes"
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"syscall"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

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
			Body:       ioutil.NopCloser(&bytes.Buffer{}),
		},
	}

	badReq = &fakeResponse{
		httpResponse: &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       ioutil.NopCloser(&bytes.Buffer{}),
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

var testBackoff = wait.Backoff{
	Steps:    5,
	Duration: 5 * time.Millisecond,
	Factor:   2.0,
	Jitter:   0.5,
	Cap:      50 * time.Millisecond,
}

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
	ctx := context.Background()
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ch := make(chan error)
			client := &testClient{responses: test.responses}
			go r.checkWithRetry(ctx, client, urltocheck, testBackoff, ch)
			if err := <-ch; (err != nil) != test.wantError {
				t.Errorf("InternetChecker.check() error = %v, wantErr %v", err, test.wantError)
			}
		})
	}
}
