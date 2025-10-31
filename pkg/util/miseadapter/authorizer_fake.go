package miseadapter

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

type fakemiseAdapter struct {
	authorized bool
	ready      bool
}

func NewFakeAuthorizer(ready bool, authorized bool) MISEAdapter {
	return &fakemiseAdapter{
		ready:      ready,
		authorized: authorized,
	}
}

func (fake *fakemiseAdapter) IsAuthorized(log *logrus.Entry, r *http.Request) (bool, error) {
	return fake.authorized, nil
}

func (fake *fakemiseAdapter) IsReady() bool {
	return fake.ready
}
