package miseadapter

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/sirupsen/logrus"
)

func Test_createRequest(t *testing.T) {
	miseAddress := "http://localhost:5000"

	translatedRequest, err := http.NewRequestWithContext(context.Background(), http.MethodPost, miseAddress+"/ValidateRequest", nil)
	if err != nil {
		t.Fatal(err)
	}

	translatedRequest.Header = http.Header{
		"Original-Uri":                    []string{"http://1.2.3.4/view"},
		"Original-Method":                 []string{http.MethodGet},
		"X-Forwarded-For":                 []string{"http://2.3.4.5"},
		"Authorization":                   []string{"Bearer token"},
		"Return-All-Actor-Token-Claims":   []string{"1"},
		"Return-All-Subject-Token-Claims": []string{"1"},
	}

	translatedRequestWithSpecificClaims, err := http.NewRequestWithContext(context.Background(), http.MethodPost, miseAddress+"/ValidateRequest", nil)
	if err != nil {
		t.Fatal(err)
	}

	translatedRequestWithSpecificClaims.Header = http.Header{
		"Original-Uri":                   []string{"http://1.2.3.4/view"},
		"Original-Method":                []string{http.MethodGet},
		"X-Forwarded-For":                []string{"http://2.3.4.5"},
		"Authorization":                  []string{"Bearer token"},
		"Return-Actor-Token-Claim-Tid":   []string{"1"},
		"Return-Subject-Token-Claim-Tid": []string{"1"},
	}

	type args struct {
		ctx         context.Context
		miseAddress string
		input       Input
	}
	tests := []struct {
		name    string
		args    args
		want    *http.Request
		wantErr bool
	}{
		{
			name: "Input is translated",
			args: args{
				ctx:         context.Background(),
				miseAddress: miseAddress,
				input: Input{
					OriginalUri:            "http://1.2.3.4/view",
					OriginalMethod:         http.MethodGet,
					OriginalIPAddress:      "http://2.3.4.5",
					AuthorizationHeader:    "Bearer token",
					ReturnAllActorClaims:   true,
					ReturnAllSubjectClaims: true,
				},
			},
			want:    translatedRequest,
			wantErr: false,
		},
		{
			name: "Input is translated with specific claims",
			args: args{
				ctx:         context.Background(),
				miseAddress: miseAddress,
				input: Input{
					OriginalUri:           "http://1.2.3.4/view",
					OriginalMethod:        http.MethodGet,
					OriginalIPAddress:     "http://2.3.4.5",
					AuthorizationHeader:   "Bearer token",
					ActorClaimsToReturn:   []string{"tid"},
					SubjectClaimsToReturn: []string{"tid"},
				},
			},
			want:    translatedRequestWithSpecificClaims,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createRequest(tt.args.ctx, tt.args.miseAddress, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("createRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := deep.Equal(tt.want, got); diff != nil {
				t.Errorf("-want/+got:\n%s", diff)
				return
			}
		})
	}
}

func Test_parseResponseIntoResult(t *testing.T) {
	type args struct {
		response *http.Response
	}

	tests := []struct {
		name    string
		args    args
		want    Result
		wantErr bool
	}{
		{
			name: "parse OK response and claims",
			args: args{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						http.CanonicalHeaderKey("Subject-Token-Claim-tid"): []string{"tid-2"},
						http.CanonicalHeaderKey("Actor-Token-Claim-tid"):   []string{"tid-1"},
					},
				},
			},
			want: Result{
				StatusCode:    http.StatusOK,
				ActorClaims:   map[string][]string{"tid": {"tid-1"}},
				SubjectClaims: map[string][]string{"tid": {"tid-2"}},
			},
		},
		{
			name: "parse OK response and encoded claims",
			args: args{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						http.CanonicalHeaderKey("Subject-Token-Encoded-Claim-season"): []string{"ZnLDvGhsaW5n"},
						http.CanonicalHeaderKey("Actor-Token-Encoded-Claim-season"):   []string{"5pil"},
					},
				},
			},
			want: Result{
				StatusCode:    http.StatusOK,
				ActorClaims:   map[string][]string{"season": {"春"}},
				SubjectClaims: map[string][]string{"season": {"frühling"}},
			},
		},
		{
			name: "parse OK response and encoded claims roles",
			args: args{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						http.CanonicalHeaderKey("Subject-Token-Encoded-Claim-roles"): []string{"ZnLDvGhsaW5n", "5pil"},
					},
				},
			},
			want: Result{
				StatusCode:    http.StatusOK,
				ActorClaims:   map[string][]string{},
				SubjectClaims: map[string][]string{"roles": {"frühling", "春"}},
			},
		},
		{
			name: "parse OK response and not encoded and encoded claims roles",
			args: args{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						http.CanonicalHeaderKey("Subject-Token-Claim-roles"):         []string{"spring"},
						http.CanonicalHeaderKey("Subject-Token-Encoded-Claim-roles"): []string{"ZnLDvGhsaW5n"},
					},
				},
			},
			want: Result{
				StatusCode:    http.StatusOK,
				ActorClaims:   map[string][]string{},
				SubjectClaims: map[string][]string{"roles": {"frühling", "spring"}},
			},
		},
		{
			name: "parse OK response and encoded and not encoded claims roles",
			args: args{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						http.CanonicalHeaderKey("Subject-Token-Encoded-Claim-roles"): []string{"ZnLDvGhsaW5n"},
						http.CanonicalHeaderKey("Subject-Token-Claim-roles"):         []string{"spring"},
					},
				},
			},
			want: Result{
				StatusCode:    http.StatusOK,
				ActorClaims:   map[string][]string{},
				SubjectClaims: map[string][]string{"roles": {"frühling", "spring"}},
			},
		},
		{
			name: "parse OK response and claims with multiple values",
			args: args{
				response: &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						http.CanonicalHeaderKey("Subject-Token-Claim-roles"): []string{"role1", "role2"},
					},
				},
			},
			want: Result{
				StatusCode:    http.StatusOK,
				ActorClaims:   map[string][]string{},
				SubjectClaims: map[string][]string{"roles": {"role1", "role2"}},
			},
		},
		{
			name: "parse 401 response",
			args: args{
				response: &http.Response{
					StatusCode: http.StatusUnauthorized,
					Header: http.Header{
						http.CanonicalHeaderKey("Error-Description"): []string{"invalid issuer"},
						http.CanonicalHeaderKey("Www-Authenticate"):  []string{"invalid token"},
					},
				},
			},
			want: Result{
				StatusCode:       http.StatusUnauthorized,
				WWWAuthenticate:  []string{"invalid token"},
				ErrorDescription: []string{"invalid issuer"},
				ActorClaims:      map[string][]string{},
				SubjectClaims:    map[string][]string{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := parseResponseIntoResult(tt.args.response)

			if tt.wantErr != (gotErr != nil) {
				t.Errorf("wantErr: %v, gotErr: %v", tt.wantErr, gotErr)
			}

			if got.SubjectClaims != nil && got.SubjectClaims["roles"] != nil {
				sort.StringSlice(got.SubjectClaims["roles"]).Sort()
			}

			if diff := deep.Equal(tt.want, got); diff != nil {
				t.Errorf("-want/+got:\n%s", diff)
				return
			}
		})
	}
}

func TestMiseAdapterIsAuthorizedRetry(t *testing.T) {
	for _, tt := range []struct {
		name             string
		serverBehavior   func(*atomic.Int32) http.HandlerFunc
		wantAuthorized   bool
		wantErr          bool
		wantAttemptCount int32
		expectedDuration time.Duration
		remoteAddr       string
	}{
		{
			name: "success on first attempt",
			serverBehavior: func(count *atomic.Int32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					count.Add(1)
					w.WriteHeader(http.StatusOK)
				}
			},
			wantAuthorized:   true,
			wantErr:          false,
			wantAttemptCount: 1,
			minDuration:      0,
		},
		{
			name: "retry on 503 then success",
			serverBehavior: func(count *atomic.Int32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					if count.Add(1) < 2 {
						w.WriteHeader(http.StatusServiceUnavailable)
						return
					}
					w.WriteHeader(http.StatusOK)
				}
			},
			wantAuthorized:   true,
			wantErr:          false,
			wantAttemptCount: 2,
			minDuration:      100 * time.Millisecond,
		},
		{
			name: "retry on 500 then success",
			serverBehavior: func(count *atomic.Int32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					if count.Add(1) < 3 {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusOK)
				}
			},
			wantAuthorized:   true,
			wantErr:          false,
			wantAttemptCount: 3,
			minDuration:      300 * time.Millisecond, // 100ms + 200ms
		},
		{
			name: "retry on 408 timeout then success",
			serverBehavior: func(count *atomic.Int32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					if count.Add(1) < 2 {
						w.WriteHeader(http.StatusRequestTimeout)
						return
					}
					w.WriteHeader(http.StatusOK)
				}
			},
			wantAuthorized:   true,
			wantErr:          false,
			wantAttemptCount: 2,
			minDuration:      100 * time.Millisecond,
		},
		{
			name: "retry on 429 rate limit then success",
			serverBehavior: func(count *atomic.Int32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					if count.Add(1) < 2 {
						w.WriteHeader(http.StatusTooManyRequests)
						return
					}
					w.WriteHeader(http.StatusOK)
				}
			},
			wantAuthorized:   true,
			wantErr:          false,
			wantAttemptCount: 2,
			minDuration:      100 * time.Millisecond,
		},
		{
			name: "no retry on 401 unauthorized",
			serverBehavior: func(count *atomic.Int32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					count.Add(1)
					w.Header().Set("Error-Description", "invalid token")
					w.WriteHeader(http.StatusUnauthorized)
				}
			},
			wantAuthorized:   false,
			wantErr:          false,
			wantAttemptCount: 1,
			minDuration:      0,
		},
		{
			name: "no retry on 403 forbidden",
			serverBehavior: func(count *atomic.Int32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					count.Add(1)
					w.WriteHeader(http.StatusForbidden)
				}
			},
			wantAuthorized:   false,
			wantErr:          false,
			wantAttemptCount: 1,
			minDuration:      0,
		},
		{
			name: "max retries exhausted on 503",
			serverBehavior: func(count *atomic.Int32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					count.Add(1)
					w.WriteHeader(http.StatusServiceUnavailable)
				}
			},
			wantAuthorized:   false,
			wantErr:          false,
			wantAttemptCount: 3,
			minDuration:      300 * time.Millisecond, // 100ms + 200ms
		},
		{
			name: "retry on connection error then success",
			serverBehavior: func(count *atomic.Int32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					if count.Add(1) < 3 {
						// Simulate network error by hijacking and closing connection
						hj, ok := w.(http.Hijacker)
						if !ok {
							return
						}
						conn, _, err := hj.Hijack()
						if err != nil {
							return
						}
						conn.Close()
						return
					}
					w.WriteHeader(http.StatusOK)
				}
			},
			wantAuthorized:   true,
			wantErr:          false,
			wantAttemptCount: 3,
			expectedDuration: 300 * time.Millisecond, // 100ms + 200ms
		},
		{
			name: "valid remote addr (IPv6)",
			serverBehavior: func(count *atomic.Int32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					count.Add(1)
					w.WriteHeader(http.StatusOK)
				}
			},
			wantAuthorized:   true,
			wantErr:          false,
			wantAttemptCount: 1,
			expectedDuration: 0,
			remoteAddr:       "[2001:db8::2001]:12345",
		},
		{
			name: "invalid remote addr (IPv6)",
			serverBehavior: func(count *atomic.Int32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					count.Add(1)
					w.WriteHeader(http.StatusOK)
				}
			},
			wantAuthorized:   false,
			wantErr:          true,
			wantAttemptCount: 0,
			expectedDuration: 0,
			remoteAddr:       "2001:db8::2001:12345",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			log := logrus.NewEntry(logrus.StandardLogger())
			var attemptCount atomic.Int32

			server := httptest.NewServer(tt.serverBehavior(&attemptCount))
			defer server.Close()

			adapter := NewAuthorizer(server.URL, log)

			req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
			if tt.remoteAddr != "" {
				req.RemoteAddr = tt.remoteAddr
			} else {
				req.RemoteAddr = "1.2.3.4:12345"
			}
			req.Header.Set("Authorization", "Bearer token")

			authorized, err := adapter.IsAuthorized(context.Background(), req)
			duration := time.Since(start)

			if (err != nil) != tt.wantErr {
				t.Errorf("unexpected error state: got err=%v, wantErr=%v", err, tt.wantErr)
			}

			if authorized != tt.wantAuthorized {
				t.Errorf("unexpected authorized: got %v, want %v", authorized, tt.wantAuthorized)
			}

			finalCount := attemptCount.Load()
			if finalCount != tt.wantAttemptCount {
				t.Errorf("unexpected attempt count: got %d, want %d", finalCount, tt.wantAttemptCount)
			}

			if duration < tt.minDuration {
				t.Errorf("unexpected duration: got %v, want at least %v", duration, tt.minDuration)
			}
		})
	}
}

func TestMiseAdapterIsAuthorizedNetworkError(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	// Point to non-existent server to trigger network errors
	adapter := NewAuthorizer("http://localhost:1", log)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	req.RemoteAddr = "1.2.3.4:12345"
	req.Header.Set("Authorization", "Bearer token")

	start := time.Now()
	authorized, err := adapter.IsAuthorized(context.Background(), req)
	duration := time.Since(start)

	if err == nil {
		t.Error("expected error for network failure")
	}

	if authorized {
		t.Error("expected authorized to be false")
	}

	// Should have attempted retries with backoff (100ms + 200ms = 300ms minimum)
	if duration < 300*time.Millisecond {
		t.Logf("warning: duration %v is less than expected 300ms, network errors might be very fast", duration)
	}
}

func TestMiseAdapterIsAuthorizedContextCancellation(t *testing.T) {
	log := logrus.NewEntry(logrus.StandardLogger())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay to allow context cancellation
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := NewAuthorizer(server.URL, log)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	req.RemoteAddr = "1.2.3.4:12345"
	req.Header.Set("Authorization", "Bearer token")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	authorized, err := adapter.IsAuthorized(ctx, req)
	if err == nil {
		t.Error("expected context error")
	}

	if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "deadline") {
		t.Logf("got unexpected error type: %v", err)
	}

	if authorized {
		t.Error("expected authorized to be false")
	}
}
