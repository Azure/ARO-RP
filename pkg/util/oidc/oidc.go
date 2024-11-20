package oidc

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"encoding/json"

	"github.com/coreos/go-oidc/v3/oidc"
)

type Verifier interface {
	Verify(context.Context, string) (Token, error)
}

type idTokenVerifier struct {
	*oidc.IDTokenVerifier
}

func (v *idTokenVerifier) Verify(ctx context.Context, rawIDToken string) (Token, error) {
	t, err := v.IDTokenVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}

	return &token{t}, nil
}

func NewVerifier(ctx context.Context, issuer, clientID string) (Verifier, error) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}

	return &idTokenVerifier{
		provider.Verifier(&oidc.Config{
			ClientID: clientID,
		}),
	}, nil
}

type Token interface {
	Claims(interface{}) error
	Subject() string
}

type token struct {
	t *oidc.IDToken
}

func (t *token) Claims(v interface{}) error {
	return t.t.Claims(v)
}

func (t *token) Subject() string {
	return t.t.Subject
}

type NoopVerifier struct {
	Err error
}

func (v *NoopVerifier) Verify(ctx context.Context, rawtoken string) (Token, error) {
	if v.Err != nil {
		return nil, v.Err
	}
	return NoopClaims(rawtoken), nil
}

type NoopClaims []byte

func (c NoopClaims) Claims(v interface{}) error {
	return json.Unmarshal(c, v)
}

func (c NoopClaims) Subject() string {
	var m map[string]interface{}
	_ = json.Unmarshal(c, &m)

	subject, _ := m["sub"].(string)

	return subject
}
