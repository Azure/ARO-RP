package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"
)

const (
	IntUsernameKey = "INT_OAUTH_USERNAME"
	IntGroupsKey   = "INT_OAUTH_GROUPS"
	IntPasswordKey = "INT_PASSWORD"
)

// IntAAD effectively disable authentication for testing purposes
type IntAAD struct {
	log            *logrus.Entry
	elevatedGroups []string
}

func NewIntAAD(groups []string, log *logrus.Entry) (IntAAD, error) {
	return IntAAD{
		elevatedGroups: groups,
		log:            log,
	}, nil
}

func (a IntAAD) Callback(w http.ResponseWriter, r *http.Request) {
}

func (a IntAAD) Login(w http.ResponseWriter, r *http.Request) {
}

func (a IntAAD) AAD(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.log.Infof("running AAD middleware from int")
		a.log.Infof("there are %d cookies", len(r.Cookies()))

		ctx := r.Context()
		ctx = context.WithValue(ctx, ContextKeyUsername, "test")
		ctx = context.WithValue(ctx, ContextKeyGroups, a.elevatedGroups)
		r = r.WithContext(ctx)

		h.ServeHTTP(w, r)
	})
}

func (a IntAAD) Logout(url string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	})
}
