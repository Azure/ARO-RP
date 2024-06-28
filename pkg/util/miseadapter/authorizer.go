package miseadapter

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

type MISEAdapter interface {
	IsAuthorized(ctx context.Context, r *http.Request) (bool, error)
}

type miseAdapter struct {
	client *Client
	log    *logrus.Entry
}

func NewAuthorizer(miseAddress string) MISEAdapter {
	return &miseAdapter{
		client: New(http.DefaultClient, miseAddress),
	}
}

func (m *miseAdapter) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	remoteAddr, _, _ := strings.Cut(r.RemoteAddr, ":")

	i := Input{
		OriginalUri:         r.RequestURI,
		OriginalMethod:      r.Method,
		OriginalIPAddress:   remoteAddr,
		AuthorizationHeader: r.Header.Get("Authorization"),
	}

	result, err := m.client.ValidateRequest(ctx, i)
	if err != nil {
		return false, err
	}

	if result.StatusCode != http.StatusOK {
		m.log.Errorf("mise authentication failed with %d: %s", result.StatusCode, result.ErrorDescription)
		return false, nil
	}

	return true, nil
}
