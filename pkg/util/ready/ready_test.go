package ready

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

type fakeHTTPClient struct {
	response    *http.Response
	responseErr error
}

func (f *fakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return f.response, f.responseErr
}

func TestURL(t *testing.T) {
	urltocheck := "http://localhost:12345/nowhere"

	type cliResp struct {
		err  error
		resp *http.Response
	}

	type test struct {
		name         string
		response     cliResp
		wantResponse bool
	}

	for _, tt := range []*test{
		{
			name:         "healthy",
			response:     cliResp{resp: &http.Response{StatusCode: 200}},
			wantResponse: true,
		},
		{
			name:         "not healthy",
			response:     cliResp{resp: &http.Response{StatusCode: 400}},
			wantResponse: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {

			cli := &fakeHTTPClient{
				response:    tt.response.resp,
				responseErr: tt.response.err,
			}

			result, err := URL(cli, urltocheck)
			if err != nil {
				t.Errorf("%s got error %s", tt.name, err.Error())
			}

			if result != tt.wantResponse {
				t.Errorf("%s: result = %v, expected %v", tt.name, result, tt.wantResponse)
			}

		})
	}
}

func TestURLPoolState(t *testing.T) {
	type cliResp struct {
		err  error
		resp *http.Response
	}

	type test struct {
		name        string
		response    cliResp
		pool        []string
		wantedState bool
		wantErr     error
	}

	for _, tt := range []*test{
		{
			name:        "healthy",
			pool:        []string{"example.com/test", "example.com/test1"},
			response:    cliResp{resp: &http.Response{StatusCode: 200}},
			wantedState: true,
		},
		{
			name:        "not healthy",
			pool:        []string{"example.com/test", "example.com/test1"},
			response:    cliResp{resp: &http.Response{StatusCode: 400}},
			wantedState: false,
		},
		{
			name:        "timeout - pool didn't propagated",
			pool:        []string{"example.com/test", "example.com/test1"},
			response:    cliResp{resp: &http.Response{StatusCode: 400}},
			wantedState: true,
			wantErr:     errors.New("timed out waiting for the condition"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cli := &fakeHTTPClient{
				response:    tt.response.resp,
				responseErr: tt.response.err,
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
			defer cancel()
			err := URLPoolState(ctx, logrus.NewEntry(logrus.StandardLogger()), cli, tt.pool, tt.wantedState)
			if err != nil {
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("%s: result = %v, expected %v", tt.name, err, tt.wantErr)
				}
			}

		})
	}
}
