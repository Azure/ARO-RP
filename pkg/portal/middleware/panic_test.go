package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	testlog "github.com/Azure/ARO-RP/test/util/log"
)

func TestPanic(t *testing.T) {
	for _, tt := range []struct {
		name      string
		panictext string
	}{
		{
			name: "ok",
		},
		{
			name:      "panic",
			panictext: "random error",
		},
	} {
		h, log := testlog.New()

		w := httptest.NewRecorder()

		Panic(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tt.panictext != "" {
				panic(tt.panictext)
			}
		})).ServeHTTP(w, nil)

		var expected []testlog.ExpectedLogEntry
		if tt.panictext == "" {
			if w.Code != http.StatusOK {
				t.Error(w.Code)
			}
		} else {
			if w.Code != http.StatusInternalServerError {
				t.Error(w.Code)
			}

			expected = []testlog.ExpectedLogEntry{
				{
					"msg":   gomega.MatchRegexp(regexp.QuoteMeta(tt.panictext)),
					"level": gomega.Equal(logrus.ErrorLevel),
				},
			}
		}

		err := testlog.AssertLoggingOutput(h, expected)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestPanicHttpAbort(t *testing.T) {
	h, log := testlog.New()

	w := httptest.NewRecorder()

	Panic(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	})).ServeHTTP(w, nil)

	if w.Code != http.StatusInternalServerError {
		t.Error(w.Code)
	}

	expected := []testlog.ExpectedLogEntry{}

	err := testlog.AssertLoggingOutput(h, expected)
	if err != nil {
		t.Error(err)
	}
}
