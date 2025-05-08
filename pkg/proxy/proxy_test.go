package proxy

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestValidation(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		subnet     string
		hostname   string
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "get https same subnet",
			method:     http.MethodGet,
			subnet:     "127.0.0.1/24",
			hostname:   "https://127.0.0.2:123",
			wantStatus: http.StatusMethodNotAllowed,
			wantErr:    true,
		},
		{
			name:       "connect http same subnet",
			method:     http.MethodConnect,
			subnet:     "127.0.0.1/24",
			hostname:   "127.0.0.2:123",
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "connect http different subnet",
			method:     http.MethodConnect,
			subnet:     "127.0.0.1/24",
			hostname:   "10.0.0.1:123",
			wantStatus: http.StatusForbidden,
			wantErr:    true,
		},
		{
			name:       "wrong hostname",
			method:     http.MethodGet,
			subnet:     "127.0.0.1/24",
			hostname:   "https://127.0.0.1::",
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := Server{Subnet: tt.subnet}
			_, subnet, err := net.ParseCIDR(server.Subnet)
			if err != nil {
				t.FailNow()
			}
			server.subnet = subnet

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.hostname, nil)

			err = server.validateProxyRequest(recorder, request)
			if (err != nil && !tt.wantErr) || (err == nil && tt.wantErr) {
				t.Error(err)
			}

			response := recorder.Result()

			if response.StatusCode != tt.wantStatus {
				fmt.Println(response.StatusCode, tt.wantStatus)
				t.Error(tt.hostname)
			}
		})
	}
}
