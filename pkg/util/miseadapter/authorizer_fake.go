package miseadapter

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"
)

type FakeMISEAdapter interface {
	IsAuthorized(ctx context.Context, r *http.Request) (bool, error)
	IsReady() bool
}

type fakemiseAdapter struct {
	client     *Client
	log        *logrus.Entry
	authorized bool
	ready      bool
}

func NewFakeAuthorizer(miseAddress string, log *logrus.Entry, fakeclient *http.Client) MISEAdapter {
	return &fakemiseAdapter{
		client: New(fakeclient, miseAddress),
		log:    log,
	}
}

func (fake *fakemiseAdapter) IsAuthorized(ctx context.Context, r *http.Request) (bool, error) {
	return fake.authorized, nil
}

func (fake *fakemiseAdapter) IsReady() bool {
	return fake.ready
}
