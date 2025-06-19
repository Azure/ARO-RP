package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"errors"
	"net/http"
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

func Panic(log *logrus.Entry) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if e := recover(); e != nil {
					if err, ok := e.(error); !ok || !errors.Is(err, http.ErrAbortHandler) {
						// ErrAbortHandler is a sentinel error that suppresses logging of a stack trace
						// https://pkg.go.dev/net/http#ErrAbortHandler
						log.Errorf("panic: %#v\n%s\n", e, string(debug.Stack()))
					}
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()

			h.ServeHTTP(w, r)
		})
	}
}
