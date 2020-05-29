package controllers

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
	"testing"

	utillog "github.com/Azure/ARO-RP/pkg/util/log"
)

type fakeClient struct {
	resp *http.Response
	err  error
}

func (c *fakeClient) Do(req *http.Request) (*http.Response, error) {
	return c.resp, c.err
}

func TestInternetCheckerCheck(t *testing.T) {
	urltocheck := "https://not-used-in-test.io"
	tests := []struct {
		name    string
		cli     *fakeClient
		wantErr bool
	}{
		{
			name: "connected",
			cli:  &fakeClient{resp: &http.Response{StatusCode: http.StatusOK}, err: nil},
		},
		{
			name: "4xx code",
			cli:  &fakeClient{resp: &http.Response{StatusCode: http.StatusBadRequest}, err: nil},
		},
		{
			name: "error",
			cli: &fakeClient{
				err: &url.Error{
					URL: urltocheck,
					Err: &net.OpError{
						Err: os.NewSyscallError("socket", syscall.ENETUNREACH),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "500 code",
			cli: &fakeClient{resp: &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Status:     http.StatusText(http.StatusServiceUnavailable),
				Body:       ioutil.NopCloser(strings.NewReader("oops sorry"))}, err: nil},
			wantErr: true,
		},
		{
			name: "timeout",
			cli: &fakeClient{resp: &http.Response{
				StatusCode: http.StatusRequestTimeout,
				Status:     http.StatusText(http.StatusRequestTimeout)},
				err: context.DeadlineExceeded},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &InternetChecker{
				Log: utillog.GetLogger(),
			}

			if err := r.check(tt.cli, urltocheck); (err != nil) != tt.wantErr {
				t.Errorf("InternetChecker.check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
