package middleware

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
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

		var expected []map[string]types.GomegaMatcher
		if tt.panictext == "" {
			if w.Code != http.StatusOK {
				t.Error(w.Code)
			}
		} else {
			if w.Code != http.StatusInternalServerError {
				t.Error(w.Code)
			}

			expected = []map[string]types.GomegaMatcher{
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
