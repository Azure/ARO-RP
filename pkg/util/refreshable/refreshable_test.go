package refreshable

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	sdkazcore "github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
)

type fakeCredential struct {
	token string
	calls int
}

// GetToken returns a long-lived token, mirroring a real credential. The
// wrapper under test is responsible for stopping azcore's BearerTokenPolicy
// from caching it for that long.
func (c *fakeCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (sdkazcore.AccessToken, error) {
	c.calls++
	return sdkazcore.AccessToken{
		Token:     c.token,
		ExpiresOn: time.Now().Add(time.Hour),
		RefreshOn: time.Now().Add(30 * time.Minute),
	}, nil
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

// TestTokenCredentialRebuildThroughBearerTokenPolicy drives a real azcore
// BearerTokenPolicy, which caches the AccessToken in its own pipeline, to prove
// Rebuild() actually changes the bearer token sent on the wire rather than only
// changing what a direct GetToken call returns.
func TestTokenCredentialRebuildThroughBearerTokenPolicy(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)
	_env := mock_env.NewMockInterface(controller)

	built := 0
	_env.EXPECT().FPNewClientCertificateCredential("tenant", nil).Times(2).DoAndReturn(
		func(tenantID string, additionalTenants []string) (azcore.TokenCredential, error) {
			built++
			return &fakeCredential{token: fmt.Sprintf("token-%d", built)}, nil
		})

	cred, err := NewFPTokenCredential(_env, "tenant", nil)
	if err != nil {
		t.Fatal(err)
	}

	var sent []string
	pipeline := runtime.NewPipeline("test", "v0.0.0", runtime.PipelineOptions{
		PerRetry: []policy.Policy{
			runtime.NewBearerTokenPolicy(cred, []string{"scope/.default"}, nil),
		},
	}, &policy.ClientOptions{
		Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
			sent = append(sent, strings.TrimPrefix(req.Header.Get("Authorization"), "Bearer "))
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
				Request:    req,
			}, nil
		}),
	})

	do := func() {
		t.Helper()
		req, err := runtime.NewRequest(ctx, http.MethodGet, "https://management.azure.com/")
		if err != nil {
			t.Fatal(err)
		}
		_, err = pipeline.Do(req)
		if err != nil {
			t.Fatal(err)
		}
	}

	// two requests before any rebuild: the same token is expected
	do()
	do()

	err = cred.Rebuild()
	if err != nil {
		t.Fatal(err)
	}

	// the pipeline must not keep serving the pre-rebuild token
	do()

	want := []string{"token-1", "token-1", "token-2"}
	if len(sent) != len(want) {
		t.Fatalf("got %v, wanted %v", sent, want)
	}
	for i := range want {
		if sent[i] != want[i] {
			t.Errorf("request %d: got %q, wanted %q", i, sent[i], want[i])
		}
	}
}

func TestTokenCredentialRebuild(t *testing.T) {
	ctx := context.Background()

	controller := gomock.NewController(t)
	_env := mock_env.NewMockInterface(controller)

	built := 0
	_env.EXPECT().FPNewClientCertificateCredential("tenant", nil).Times(2).DoAndReturn(
		func(tenantID string, additionalTenants []string) (azcore.TokenCredential, error) {
			built++
			return &fakeCredential{token: fmt.Sprintf("token-%d", built)}, nil
		})

	cred, err := NewFPTokenCredential(_env, "tenant", nil)
	if err != nil {
		t.Fatal(err)
	}

	token, err := cred.GetToken(ctx, policy.TokenRequestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if token.Token != "token-1" {
		t.Errorf("got %q, wanted token-1", token.Token)
	}

	// without a rebuild, the same underlying credential is used
	token, err = cred.GetToken(ctx, policy.TokenRequestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if token.Token != "token-1" {
		t.Errorf("got %q, wanted token-1", token.Token)
	}

	err = cred.Rebuild()
	if err != nil {
		t.Fatal(err)
	}

	token, err = cred.GetToken(ctx, policy.TokenRequestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if token.Token != "token-2" {
		t.Errorf("got %q, wanted token-2", token.Token)
	}
}

func TestTokenCredentialRebuildError(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("cannot build credential")

	controller := gomock.NewController(t)
	_env := mock_env.NewMockInterface(controller)

	gomock.InOrder(
		_env.EXPECT().FPNewClientCertificateCredential("tenant", nil).Return(&fakeCredential{token: "token"}, nil),
		_env.EXPECT().FPNewClientCertificateCredential("tenant", nil).Return(nil, wantErr),
	)

	cred, err := NewFPTokenCredential(_env, "tenant", nil)
	if err != nil {
		t.Fatal(err)
	}

	err = cred.Rebuild()
	if !errors.Is(err, wantErr) {
		t.Errorf("got %v, wanted %v", err, wantErr)
	}

	// a failed rebuild must leave the existing credential usable
	token, err := cred.GetToken(ctx, policy.TokenRequestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if token.Token != "token" {
		t.Errorf("got %q, wanted token", token.Token)
	}
}

func TestNewFPTokenCredentialError(t *testing.T) {
	wantErr := errors.New("cannot build credential")

	controller := gomock.NewController(t)
	_env := mock_env.NewMockInterface(controller)
	_env.EXPECT().FPNewClientCertificateCredential("tenant", nil).Return(nil, wantErr)

	_, err := NewFPTokenCredential(_env, "tenant", nil)
	if !errors.Is(err, wantErr) {
		t.Errorf("got %v, wanted %v", err, wantErr)
	}
}

type fakeRebuilder struct {
	calls int
	err   error
}

func (r *fakeRebuilder) Rebuild() error {
	r.calls++
	return r.err
}

func TestMultiRebuilder(t *testing.T) {
	wantErr := errors.New("boom")

	for _, tt := range []struct {
		name      string
		first     *fakeRebuilder
		second    *fakeRebuilder
		wantErr   error
		wantCalls []int
	}{
		{
			name:      "rebuilds all",
			first:     &fakeRebuilder{},
			second:    &fakeRebuilder{},
			wantCalls: []int{1, 1},
		},
		{
			name:      "stops at first error",
			first:     &fakeRebuilder{err: wantErr},
			second:    &fakeRebuilder{},
			wantErr:   wantErr,
			wantCalls: []int{1, 0},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := NewMultiRebuilder(tt.first, nil, tt.second).Rebuild()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("got %v, wanted %v", err, tt.wantErr)
			}
			if tt.first.calls != tt.wantCalls[0] || tt.second.calls != tt.wantCalls[1] {
				t.Errorf("got calls %d,%d wanted %v", tt.first.calls, tt.second.calls, tt.wantCalls)
			}
		})
	}
}
