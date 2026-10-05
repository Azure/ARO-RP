package refreshable

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"

	sdkazcore "github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"

	"github.com/Azure/ARO-RP/pkg/util/azureclient/azuresdk/azcore"
	mock_env "github.com/Azure/ARO-RP/pkg/util/mocks/env"
)

type fakeCredential struct {
	token string
}

func (c *fakeCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (sdkazcore.AccessToken, error) {
	return sdkazcore.AccessToken{Token: c.token}, nil
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
